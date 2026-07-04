package actor

import (
	"bytes"
	"context"
	"encoding/json"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/alfreddobradi/actors/cmd/game/game"
	"github.com/alfreddobradi/actors/cmd/game/model"
	"github.com/alfreddobradi/actors/pkg/database/kv/memory"
	"github.com/alfreddobradi/actors/pkg/system"
	"github.com/alfreddobradi/actors/pkg/testhelper"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

const (
	accountName   = "TestAccount"
	characterName = "TestCharacter"
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
	sys := system.MustNewSystem(registry, nil, db)

	params := model.AccountActorParams{Name: accountName}
	actorHandler, err := sys.SpawnWithParams(ctx, "AccountActor", params)
	require.NoError(t, err)
	require.NotNil(t, actorHandler)

	actor := actorHandler.GetActor().(*AccountActor)
	require.Equal(t, accountName, actor.Name)
	require.Nil(t, actor.Guild)
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
			params:       model.AccountActorParams{ID: fooID, Name: accountName},
			testID:       func(id uuid.UUID) bool { return id == fooID },
			expectedName: accountName,
		},
		{
			label:        "IDParams",
			params:       ID{ID: barID},
			testID:       func(id uuid.UUID) bool { return id == barID },
			expectedName: stringDefault,
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
			expectedName: stringDefault,
		},
		{
			label:        "NilParams",
			params:       nil,
			testID:       func(id uuid.UUID) bool { return id != uuid.Nil },
			expectedName: stringDefault,
		},
	}

	ctx := context.Background()
	registry := system.NewRegistry()
	registry.RegisterFactory("AccountActor", accountActorFactory)

	db := memory.NewStore()
	sys := system.MustNewSystem(registry, nil, db)

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
	sys := system.MustNewSystem(registry, nil, db)
	params := model.AccountActorParams{Name: "PersistentAccount"}
	actorHandler, err := sys.SpawnWithParams(ctx, "AccountActor", params)
	require.NoError(t, err)
	require.NotNil(t, actorHandler)

	character := &game.Hero{
		ID:   uuid.New(),
		Name: characterName,
		Action: &game.GatherAction{
			Resource: game.Wood,
		},
	}

	character.GainExperience(1000)
	character.Status = game.StatusBusy
	character.Cooldown = 3

	actor := actorHandler.GetActor().(*AccountActor)
	actor.Guild = game.NewGuild("PersistentTavern")
	actor.Guild.AddCharacter(character)

	snapshot, err := actorHandler.GetActor().Snapshot(ctx)
	require.NoError(t, err)

	restoredAccount := &AccountActor{}
	err = restoredAccount.RestoreFromSnapshot(ctx, snapshot)

	require.NoError(t, err)
	require.Equal(t, "PersistentAccount", restoredAccount.Name)
	require.NotNil(t, restoredAccount.Guild)

	gold := restoredAccount.Guild.Gold.Load()
	require.Equal(t, int64(3000), gold)

	char, exists := restoredAccount.Guild.GetCharacter(character.ID)
	require.True(t, exists)
	require.Equal(t, 1000, char.Experience)
	require.Equal(t, characterName, char.Name)
	require.Equal(t, game.StatusBusy, char.Status)
	require.Equal(t, 3, char.Cooldown)
	require.NotNil(t, char.Action)
	require.IsType(t, &game.GatherAction{}, char.Action)
	gatherAction := char.Action.(*game.GatherAction)
	require.Equal(t, game.Wood, gatherAction.Resource)
}

func TestAccountJSONRoundTrip(t *testing.T) {
	gold := &atomic.Int64{}
	gold.Store(1000)

	testHeroWithAction := &game.Hero{
		ID:   uuid.New(),
		Name: characterName,
		Action: &game.GatherAction{
			Resource: game.Wood,
		},
		Level:      5,
		Experience: 1500,
		Status:     game.StatusBusy,
		Cooldown:   2,
	}

	testHeroNoAction := &game.Hero{
		ID:         uuid.New(),
		Name:       characterName,
		Action:     nil,
		Level:      5,
		Experience: 1500,
		Status:     game.StatusBusy,
		Cooldown:   2,
	}

	tests := []struct {
		label         string
		guild         *game.Guild
		hero          *game.Hero
		expectedChars map[uuid.UUID]*game.Hero
	}{
		{
			label: "Account with Tavern and Hero with an Action",
			guild: func() *game.Guild {
				t := game.NewGuild("TestTavern")
				t.AddCharacter(testHeroWithAction)
				return t
			}(),
			hero: testHeroWithAction,
			expectedChars: map[uuid.UUID]*game.Hero{
				testHeroWithAction.ID: testHeroWithAction,
			},
		},
		{
			label: "Account with Tavern and Hero with no Action",
			guild: func() *game.Guild {
				t := game.NewGuild("TestTavern")
				t.AddCharacter(testHeroNoAction)
				return t
			}(),
			hero: testHeroNoAction,
			expectedChars: map[uuid.UUID]*game.Hero{
				testHeroNoAction.ID: testHeroNoAction,
			},
		},
		{
			label:         "Account with Empty Tavern",
			guild:         game.NewGuild("EmptyTavern"),
			expectedChars: map[uuid.UUID]*game.Hero{},
		},
		{
			label: "Account with No Tavern",
			guild: nil,
		},
	}

	for _, tt := range tests {
		tf := func(t *testing.T) {
			tg := tt.guild
			if tg != nil {
				tg.Gold = gold
			}

			account := &AccountActor{
				ID:    uuid.New(),
				Name:  accountName,
				Guild: tg,
			}

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

			if len(tt.expectedChars) > 0 {
				char, exists := unmarshaledAccount.Guild.GetCharacter(tt.hero.ID)
				require.True(t, exists)
				require.Equal(t, tt.hero.Name, char.Name)
				require.Equal(t, tt.hero.Level, char.Level)
				require.Equal(t, tt.hero.Experience, char.Experience)
				require.Equal(t, tt.hero.Status, char.Status)
				require.Equal(t, tt.hero.Cooldown, char.Cooldown)
				if tt.hero.Action == nil {
					require.Nil(t, char.Action)
				} else {
					require.Equal(t, tt.hero.Action.GetName(), char.Action.GetName())
				}
			}

			if tt.guild != nil {
				// Gold should be preserved through JSON round trip
				goldValue := unmarshaledAccount.Guild.Gold.Load()
				require.Equal(t, int64(1000), goldValue)
				require.Equal(t, tt.guild.Name(), unmarshaledAccount.Guild.Name())
				require.Equal(t, len(tt.expectedChars), len(unmarshaledAccount.Guild.Characters()))
			} else {
				require.Nil(t, unmarshaledAccount.Guild)
			}
		}
		t.Run(tt.label, tf)
	}
}

func TestAccountUnmarshalJSON(t *testing.T) {
	account := &AccountActor{
		ID:    uuid.New(),
		Name:  accountName,
		Guild: game.NewGuild("TestTavern"),
	}

	inventory := game.NewInventory()
	inventory.AddResource(game.Wood, 10)

	character := &game.Hero{
		ID:         uuid.New(),
		Name:       characterName,
		Level:      5,
		Experience: 1500,
		Status:     game.StatusBusy,
		Cooldown:   2,
		Action: &game.GatherAction{
			Resource: game.Wood,
		},
		Inventory: inventory,
	}

	account.Guild.AddCharacter(character)

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

	char, exists := unmarshaledAccount.Guild.GetCharacter(character.ID)
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
	require.Equal(t, account.Guild.Name(), unmarshaledAccount.Guild.Name())
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
				mx:    &sync.Mutex{},
				ID:    uuid.New(),
				Name:  accountName,
				Guild: nil,
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
				require.Nil(t, account.Guild)
			} else {
				require.NoError(t, err)
				require.NotNil(t, account.Guild)
				require.Equal(t, tt.tavernName, account.Guild.Name())
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
				mx:    &sync.Mutex{},
				ID:    uuid.New(),
				Name:  accountName,
				Guild: game.NewGuild("TestTavern"),
			}
			account.Guild.Gold = gold

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
				require.Equal(t, tt.gold, account.Guild.Gold.Load())
				require.Equal(t, 0, len(account.Guild.Characters()))
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.gold-1000, account.Guild.Gold.Load())
				require.Equal(t, 1, len(account.Guild.Characters()))
			}
		}
		t.Run(tt.label, tf)
	}
}
