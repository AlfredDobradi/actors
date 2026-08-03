package actor

import (
	"context"
	"testing"

	"github.com/alfreddobradi/actors/cmd/game/repository/mock"
	"github.com/alfreddobradi/actors/pkg/database/kv/memory"
	"github.com/alfreddobradi/actors/pkg/model"
	"github.com/alfreddobradi/actors/pkg/system"
	"github.com/alfreddobradi/actors/pkg/testhelper"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestKeeperActorFactory(t *testing.T) {
	kind := "KeeperActor"
	repo := mock.New()

	testhelper.SetupTestLogger(false)

	registry := system.NewRegistry()
	registry.RegisterFactory(kind, keeperActorFactory)

	db := memory.NewStore()

	mockSendFunc := func(
		ctx context.Context,
		expectsResponse bool,
		senderID uuid.UUID,
		recipient system.Recipient,
		message any,
	) (any, error) {
		return nil, nil
	}

	ctx := context.WithValue(context.Background(), model.ContextKeyDBHandle, repo)
	ctxWithSender := context.WithValue(ctx, model.ContextKeySenderFn, mockSendFunc)

	sys := system.MustNewSystem(registry, nil, db)
	actorHandler, err := sys.Spawn(ctxWithSender, kind)
	require.NoError(t, err)
	require.NotNil(t, actorHandler)

	actor := actorHandler.GetActor().(*KeeperActor)
	require.NotEqual(t, uuid.Nil, actor.ID)
	require.NotNil(t, actor.handle)
	require.NotNil(t, actor.sendCallback)
	require.Equal(t, kind, actor.GetKind())

	_, ok := actor.handle.(*mock.Repository)
	require.True(t, ok)
}
