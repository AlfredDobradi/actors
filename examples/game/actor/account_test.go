package actor

import (
	"bytes"
	"context"
	"encoding/json"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alfreddobradi/actors/examples/game/game"
	"github.com/alfreddobradi/actors/examples/game/model"
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
	actor.Tavern = game.NewTavern()
	actor.Tavern.AddCharacter(character)

	snapshot, err := actorHandler.GetActor().Snapshot(ctx)
	require.NoError(t, err)

	restoredAccount := &AccountActor{}
	err = restoredAccount.RestoreFromSnapshot(ctx, snapshot)

	require.NoError(t, err)
	require.Equal(t, "PersistentAccount", restoredAccount.Name)
	require.NotNil(t, restoredAccount.Tavern)

	gold := restoredAccount.Gold.Load()
	require.Equal(t, uint64(3000), gold)

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
		Tavern: game.NewTavern(),
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

	actor.replayTicks(ctx, since)
	character, exists := actor.Tavern.GetCharacter(character.ID)
	require.True(t, exists)

	require.Equal(t, 3, character.Inventory.GetResource(resource))
	require.Equal(t, 30, character.Experience)
}

func TestAccountJSONRoundTrip(t *testing.T) {
	gold := &atomic.Uint64{}
	gold.Store(1000)

	account := &AccountActor{
		ID:     uuid.New(),
		Name:   "TestAccount",
		Tavern: game.NewTavern(),
		Gold:   gold,
	}

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
	}

	account.Tavern.AddCharacter(character)

	raw, err := json.Marshal(account)
	require.NoError(t, err)

	var unmarshaledAccount AccountActor
	err = json.Unmarshal(raw, &unmarshaledAccount)
	require.NoError(t, err)

	require.Equal(t, account.ID, unmarshaledAccount.ID)
	require.Equal(t, account.Name, unmarshaledAccount.Name)
	require.NotNil(t, unmarshaledAccount.Tavern)

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

	// Gold should be preserved through JSON round trip
	goldValue := unmarshaledAccount.Gold.Load()
	require.Equal(t, uint64(1000), goldValue)
}

func TestAccountUnmarshalJSON(t *testing.T) {
	account := &AccountActor{
		ID:     uuid.New(),
		Name:   "TestAccount",
		Tavern: game.NewTavern(),
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
}

func TestAccountCreateTavern(t *testing.T) {
	account := &AccountActor{
		ID:     uuid.New(),
		Name:   "TestAccount",
		Tavern: nil,
	}

	require.Nil(t, account.Tavern)
	ctx := context.Background()

	createMessage := &system.Message{
		ID:        uuid.New(),
		Sender:    uuid.Nil,
		Payload:   model.NewTavernRequest{},
		Recipient: system.Recipient{Kind: system.RecipientKindActor, Subject: account.ID.String()},
	}

	account.createTavern(ctx, createMessage)
	require.NotNil(t, account.Tavern)

	account.Tavern.AddCharacter(&game.Character{
		ID: uuid.New(),
	})

	// Creating tavern again should not overwrite existing tavern
	account.createTavern(ctx, createMessage)
	require.NotNil(t, account.Tavern)
}
