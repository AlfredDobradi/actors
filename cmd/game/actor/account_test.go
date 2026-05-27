package actor

import (
	"bytes"
	"context"
	"encoding/json"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alfreddobradi/actors/cmd/game/game"
	"github.com/alfreddobradi/actors/cmd/game/model"
	"github.com/alfreddobradi/actors/pkg/database/memory"
	"github.com/alfreddobradi/actors/pkg/system"
	"github.com/alfreddobradi/actors/pkg/testhelper"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type ID struct {
	ID uuid.UUID
}

func (id ID) GetID() uuid.UUID {
	return id.ID
}

func TestNewAccountHasNoTavern(t *testing.T) {
	testhelper.SetupTestLogger(testing.Verbose())
	ctx := context.Background()
	registry := system.NewRegistry()
	registry.RegisterFactory("AccountActor", accountActorFactory)
	db := memory.NewStore()
	sys := system.MustNewSystem(registry, db)

	params := model.AccountActorParams{Name: "TestAccount"}
	actorHandler, err := sys.SpawnWithParams(ctx, "AccountActor", params)
	require.NoError(t, err)
	require.NotNil(t, actorHandler)

	actor := actorHandler.GetActor().(*AccountActor)
	require.Equal(t, "TestAccount", actor.Name)
	require.Nil(t, actor.Tavern)
}

func TestAccountActorFactory(t *testing.T) {
	fooID := uuid.New()
	barID := uuid.New()

	type idTest func(id uuid.UUID) bool

	tests := []struct {
		label        string
		params       any
		testID       idTest
		expectedName string
	}{
		{
			label:        "AccountActorParams",
			params:       model.AccountActorParams{ID: fooID, Name: "TestAccount"},
			testID:       func(id uuid.UUID) bool { return id == fooID },
			expectedName: "TestAccount",
		},
		{
			label:        "IDParams",
			params:       ID{ID: barID},
			testID:       func(id uuid.UUID) bool { return id == barID },
			expectedName: "default",
		},
		{
			label: "InvalidParams",
			params: struct {
				Foo string
			}{
				Foo: "invalid",
			},
			// when param type is something unhandled, factory should generate a random ID
			testID:       func(id uuid.UUID) bool { return id != uuid.Nil },
			expectedName: "default",
		},
		{
			label:        "NilParams",
			params:       nil,
			testID:       func(id uuid.UUID) bool { return id != uuid.Nil },
			expectedName: "default",
		},
	}

	ctx := context.Background()
	registry := system.NewRegistry()
	registry.RegisterFactory("AccountActor", accountActorFactory)

	db := memory.NewStore()
	sys := system.MustNewSystem(registry, db)

	for _, tt := range tests {
		tf := func(t *testing.T) {
			params := tt.params
			actorHandler, err := sys.SpawnWithParams(ctx, "AccountActor", params)
			require.NoError(t, err)
			require.NotNil(t, actorHandler)

			actor := actorHandler.GetActor().(*AccountActor)
			require.Equal(t, tt.expectedName, actor.Name)
			require.True(t, tt.testID(actor.ID))
		}
		t.Run(tt.label, tf)
	}
}

func TestActorPersistence(t *testing.T) {
	ctx := context.Background()
	registry := system.NewRegistry()
	registry.RegisterFactory("AccountActor", accountActorFactory)
	db := memory.NewStore()
	sys := system.MustNewSystem(registry, db)
	params := model.AccountActorParams{Name: "PersistentAccount"}
	actorHandler, err := sys.SpawnWithParams(ctx, "AccountActor", params)
	require.NoError(t, err)
	require.NotNil(t, actorHandler)

	character := &game.Character{
		ID:   uuid.New(),
		Name: "TestCharacter",
		Action: &game.GatherAction{
			Resource: game.Wood,
		},
	}

	character.GainExperience(1000)
	character.Status = game.StatusBusy
	character.Cooldown = 3

	actor := actorHandler.GetActor().(*AccountActor)
	actor.Tavern = game.NewTavern("PersistentTavern")
	actor.Tavern.AddCharacter(character)

	snapshot, err := actorHandler.GetActor().Snapshot(ctx)
	require.NoError(t, err)

	restoredAccount := &AccountActor{}
	err = restoredAccount.RestoreFromSnapshot(ctx, snapshot)

	require.NoError(t, err)
	require.Equal(t, "PersistentAccount", restoredAccount.Name)
	require.NotNil(t, restoredAccount.Tavern)

	gold := restoredAccount.Gold.Load()
	require.Equal(t, int64(3000), gold)

	char, exists := restoredAccount.Tavern.GetCharacter(character.ID)
	require.True(t, exists)
	require.Equal(t, 1000, char.Experience)
	require.Equal(t, "TestCharacter", char.Name)
	require.Equal(t, game.StatusBusy, char.Status)
	require.Equal(t, 3, char.Cooldown)
	require.NotNil(t, char.Action)
	require.IsType(t, &game.GatherAction{}, char.Action)
	gatherAction := char.Action.(*game.GatherAction)
	require.Equal(t, game.Wood, gatherAction.Resource)
}

func TestReplayTicks(t *testing.T) {
	testhelper.SetupTestLogger(testing.Verbose())
	resource := game.Resource{Name: "test_resource", Experience: 10, Difficulty: 0.0, CooldownMultiplier: 1.0, BatchSize: [2]int{1, 1}}
	ctx := context.Background()
	actor := &AccountActor{
		ID:     uuid.New(),
		Name:   "TestAccount",
		Tavern: game.NewTavern("TestTavern"),
	}
	character := &game.Character{
		ID:   uuid.New(),
		Name: "TestCharacter",
		Action: &game.GatherAction{
			Resource: resource,
		},
		Experience: 0,
		Inventory:  game.NewInventory(),
	}
	actor.Tavern.AddCharacter(character)

	since := time.Now().Add(-15 * time.Second).Unix()

	require.Equal(t, 0, character.Experience)
	require.Equal(t, 0, character.Inventory.GetResource(resource))

	err := actor.replayTicks(ctx, since)
	require.NoError(t, err)
	character, exists := actor.Tavern.GetCharacter(character.ID)
	require.True(t, exists)

	require.Equal(t, 3, character.Inventory.GetResource(resource))
	require.Equal(t, 30, character.Experience)
}

func TestAccountJSONRoundTrip(t *testing.T) {
	gold := &atomic.Int64{}
	gold.Store(1000)

	testHero := &game.Character{
		ID:   uuid.New(),
		Name: "TestCharacter",
		Action: &game.GatherAction{
			Resource: game.Wood,
		},
		Level:      5,
		Experience: 1500,
		Status:     game.StatusBusy,
		Cooldown:   2,
	}

	tests := []struct {
		label         string
		tavern        *game.Tavern
		expectedChars map[uuid.UUID]*game.Character
	}{
		{
			label: "Account with Tavern and Characters",
			tavern: func() *game.Tavern {
				t := game.NewTavern("TestTavern")
				t.AddCharacter(testHero)
				return t
			}(),
			expectedChars: map[uuid.UUID]*game.Character{
				testHero.ID: testHero,
			},
		},
		{
			label:         "Account with Empty Tavern",
			tavern:        game.NewTavern("EmptyTavern"),
			expectedChars: map[uuid.UUID]*game.Character{},
		},
		{
			label:  "Account with No Tavern",
			tavern: nil,
		},
	}

	for _, tt := range tests {
		tf := func(t *testing.T) {
			account := &AccountActor{
				ID:     uuid.New(),
				Name:   "TestAccount",
				Tavern: tt.tavern,
				Gold:   gold,
			}

			raw, err := json.Marshal(account)
			require.NoError(t, err)

			var unmarshaledAccount AccountActor
			err = json.Unmarshal(raw, &unmarshaledAccount)
			require.NoError(t, err)

			require.Equal(t, account.ID, unmarshaledAccount.ID)
			require.Equal(t, account.Name, unmarshaledAccount.Name)

			if len(tt.expectedChars) > 0 {
				char, exists := unmarshaledAccount.Tavern.GetCharacter(testHero.ID)
				require.True(t, exists)
				require.Equal(t, testHero.Name, char.Name)
				require.Equal(t, testHero.Level, char.Level)
				require.Equal(t, testHero.Experience, char.Experience)
				require.Equal(t, testHero.Status, char.Status)
				require.Equal(t, testHero.Cooldown, char.Cooldown)
				require.NotNil(t, char.Action)
				require.IsType(t, &game.GatherAction{}, char.Action)
				gatherAction := char.Action.(*game.GatherAction)
				require.Equal(t, game.Wood, gatherAction.Resource)
			}

			// Gold should be preserved through JSON round trip
			goldValue := unmarshaledAccount.Gold.Load()
			require.Equal(t, int64(1000), goldValue)

			if tt.tavern != nil {
				require.Equal(t, tt.tavern.Name(), unmarshaledAccount.Tavern.Name())
				require.Equal(t, len(tt.expectedChars), len(unmarshaledAccount.Tavern.Characters()))
			} else {
				require.Nil(t, unmarshaledAccount.Tavern)
			}
		}
		t.Run(tt.label, tf)
	}
}

func TestAccountUnmarshalJSON(t *testing.T) {
	account := &AccountActor{
		ID:     uuid.New(),
		Name:   "TestAccount",
		Tavern: game.NewTavern("TestTavern"),
	}

	inventory := game.NewInventory()
	inventory.AddResource(game.Wood, 10)

	character := &game.Character{
		ID:         uuid.New(),
		Name:       "TestCharacter",
		Level:      5,
		Experience: 1500,
		Status:     game.StatusBusy,
		Cooldown:   2,
		Action: &game.GatherAction{
			Resource: game.Wood,
		},
		Inventory: inventory,
	}

	account.Tavern.AddCharacter(character)

	buf := bytes.NewBufferString("")
	encoder := json.NewEncoder(buf)
	encoder.SetIndent("", "  ")

	err := encoder.Encode(account)
	require.NoError(t, err)

	var unmarshaledAccount AccountActor
	err = json.Unmarshal(buf.Bytes(), &unmarshaledAccount)
	require.NoError(t, err)

	require.Equal(t, account.ID, unmarshaledAccount.ID)
	require.Equal(t, account.Name, unmarshaledAccount.Name)

	char, exists := unmarshaledAccount.Tavern.GetCharacter(character.ID)
	require.True(t, exists)
	require.Equal(t, character.Name, char.Name)
	require.Equal(t, character.Level, char.Level)
	require.Equal(t, character.Experience, char.Experience)
	require.Equal(t, character.Status, char.Status)
	require.Equal(t, character.Cooldown, char.Cooldown)
	require.NotNil(t, char.Action)
	require.IsType(t, &game.GatherAction{}, char.Action)
	gatherAction := char.Action.(*game.GatherAction)
	require.Equal(t, game.Wood, gatherAction.Resource)
	require.NotNil(t, char.Inventory)
	require.Equal(t, 10, char.Inventory.GetResource(game.Wood))
	require.Equal(t, account.Tavern.Name(), unmarshaledAccount.Tavern.Name())
}

func TestAccountCreateTavern(t *testing.T) {
	tests := []struct {
		label       string
		tavernName  string
		expectError bool
	}{
		{
			label:       "Valid Tavern Name",
			tavernName:  "MyTavern",
			expectError: false,
		},
		{
			label:       "Empty Tavern Name",
			tavernName:  "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		tf := func(t *testing.T) {
			account := &AccountActor{
				mx:     &sync.Mutex{},
				ID:     uuid.New(),
				Name:   "TestAccount",
				Tavern: nil,
			}

			ctx := context.Background()
			createMessage := &system.Message{
				ID:        uuid.New(),
				Sender:    uuid.Nil,
				Payload:   model.NewTavernRequest{Name: tt.tavernName},
				Recipient: system.Recipient{Kind: system.RecipientKindActor, Subject: account.ID.String()},
			}

			err := account.createTavern(ctx, createMessage)
			if tt.expectError {
				require.Error(t, err)
				require.Nil(t, account.Tavern)
			} else {
				require.NoError(t, err)
				require.NotNil(t, account.Tavern)
				require.Equal(t, tt.tavernName, account.Tavern.Name())
			}
		}
		t.Run(tt.label, tf)
	}
}

func TestAccountHireCharacter(t *testing.T) {
	tests := []struct {
		label       string
		gold        int64
		expectError bool
	}{
		{
			label:       "Sufficient Gold",
			gold:        5000,
			expectError: false,
		},
		{
			label:       "Insufficient Gold",
			gold:        1,
			expectError: true,
		},
	}

	for _, tt := range tests {
		tf := func(t *testing.T) {
			gold := &atomic.Int64{}
			gold.Store(tt.gold)

			account := &AccountActor{
				mx:     &sync.Mutex{},
				ID:     uuid.New(),
				Name:   "TestAccount",
				Tavern: game.NewTavern("TestTavern"),
				Gold:   gold,
			}

			ctx := context.Background()
			hireMessage := &system.Message{
				ID:        uuid.New(),
				Sender:    uuid.Nil,
				Payload:   model.HireCharacterRequest{},
				Recipient: system.Recipient{Kind: system.RecipientKindActor, Subject: account.ID.String()},
			}

			err := account.hireCharacter(ctx, hireMessage)
			if tt.expectError {
				require.Error(t, err)
				require.Equal(t, tt.gold, account.Gold.Load())
				require.Equal(t, 0, len(account.Tavern.Characters()))
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.gold-1000, account.Gold.Load())
				require.Equal(t, 1, len(account.Tavern.Characters()))
			}
		}
		t.Run(tt.label, tf)
	}
}
