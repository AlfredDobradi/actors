package system_test

import (
	"context"
	"testing"

	"github.com/alfreddobradi/actors/pkg/system"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestRegistrySpawn(t *testing.T) {
	registry := system.NewRegistry()
	// Register a simple actor factory
	registry.RegisterFactory("testActor", func(ctx context.Context) system.Actor {
		return &MockActor{id: uuid.New(), kind: "testActor"}
	})

	bus := system.NewBus()
	sys := system.NewSystem(bus, registry)

	ctx := context.Background()
	handler, err := sys.Spawn(ctx, "testActor")
	require.NoError(t, err)
	require.NotNil(t, handler)
	require.Equal(t, "testActor", handler.GetActor().GetKind())

	// Clean up
	handler.Stop()
}
