package actor

import (
	"context"
	"testing"

	"github.com/alfreddobradi/actors/pkg/system"
	"github.com/stretchr/testify/require"
)

func TestAccountActorFactory(t *testing.T) {
	ctx := context.Background()
	registry := system.NewRegistry()
	registry.RegisterFactory("AccountActor", accountActorFactory)

	sys := system.MustNewSystem(registry, nil)
	params := AccountActorParams{Name: "TestAccount"}
	actorHandler, err := sys.SpawnWithParams(ctx, "AccountActor", params)
	require.NoError(t, err)
	require.NotNil(t, actorHandler)

	actor := actorHandler.GetActor().(*AccountActor)
	require.Equal(t, "TestAccount", actor.Name)
}
