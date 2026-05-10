package actor

import (
	"context"
	"encoding/json"
	"sync"
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

	db, err := memory.NewStore()
	require.NoError(t, err)
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
	db, err := memory.NewStore()
	require.NoError(t, err)
	sys := system.MustNewSystem(registry, db)
	params := model.AccountActorParams{Name: "PersistentAccount"}
	actorHandler, err := sys.SpawnWithParams(ctx, "AccountActor", params)
	require.NoError(t, err)
	require.NotNil(t, actorHandler)

	character := game.Character{
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
	actor.Characters.mx.Lock()
	actor.Characters.characters[character.ID] = character
	actor.Characters.mx.Unlock()

	snapshot, err := actorHandler.GetActor().Snapshot(ctx)
	require.NoError(t, err)

	restoredAccount := &AccountActor{}
	err = restoredAccount.RestoreFromSnapshot(ctx, snapshot)

	require.NoError(t, err)
	require.Equal(t, "PersistentAccount", restoredAccount.Name)
	require.NotNil(t, restoredAccount.Characters)

	char := restoredAccount.Characters.characters[character.ID]
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
		ID:         uuid.New(),
		Name:       "TestAccount",
		Characters: newCharacterStore(),
	}
	character := game.Character{
		ID:   uuid.New(),
		Name: "TestCharacter",
		Action: &game.GatherAction{
			Resource: resource,
		},
		Experience: 0,
		Inventory:  game.NewInventory(),
	}
	actor.Characters.characters[character.ID] = character

	since := time.Now().Add(-15 * time.Second).Unix()

	require.Equal(t, 0, character.Experience)
	require.Equal(t, 0, character.Inventory.GetResource(resource))

	actor.ReplayTicks(ctx, since)
	character = actor.Characters.characters[character.ID]

	require.Equal(t, 3, character.Inventory.GetResource(resource))
	require.Equal(t, 30, character.Experience)
}

func TestAccountJSONRoundTrip(t *testing.T) {
	account := &AccountActor{
		ID:   uuid.New(),
		Name: "TestAccount",
		Characters: &characterStore{
			mx: &sync.Mutex{},
			characters: map[uuid.UUID]game.Character{
				uuid.New(): {
					ID:         uuid.New(),
					Name:       "TestCharacter",
					Level:      5,
					Experience: 1500,
					Status:     game.StatusBusy,
					Cooldown:   2,
					Action: &game.GatherAction{
						Resource: game.Wood,
					},
				},
			},
		},
	}

	raw, err := json.Marshal(account)
	require.NoError(t, err)

	var unmarshaledAccount AccountActor
	err = json.Unmarshal(raw, &unmarshaledAccount)
	require.NoError(t, err)
}
