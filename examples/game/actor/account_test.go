package actor

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/alfreddobradi/actors/examples/game/game"
	"github.com/alfreddobradi/actors/examples/game/model"
	"github.com/alfreddobradi/actors/pkg/database/memory"
	"github.com/alfreddobradi/actors/pkg/system"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestAccountActorFactory(t *testing.T) {
	ctx := context.Background()
	registry := system.NewRegistry()
	registry.RegisterFactory("AccountActor", accountActorFactory)

	db, err := memory.NewStore()
	require.NoError(t, err)
	sys := system.MustNewSystem(registry, db)
	params := model.AccountActorParams{Name: "TestAccount"}
	actorHandler, err := sys.SpawnWithParams(ctx, "AccountActor", params)
	require.NoError(t, err)
	require.NotNil(t, actorHandler)

	actor := actorHandler.GetActor().(*AccountActor)
	require.Equal(t, "TestAccount", actor.Name)
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

	err = actor.Persist(ctx, db)
	require.NoError(t, err)

	key := fmt.Sprintf("actor:account:%s", actor.ID)
	persistedActor, ok := db.Get(ctx, key)
	require.True(t, ok)
	require.NotNil(t, persistedActor)

	var restoredAccount *AccountActor
	err = json.Unmarshal([]byte(persistedActor), &restoredAccount)
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
