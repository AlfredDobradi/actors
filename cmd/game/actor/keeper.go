package actor

import (
	"context"
	"log/slog"
	"time"

	"github.com/alfreddobradi/actors/cmd/game/game"
	"github.com/alfreddobradi/actors/cmd/game/repository"
	"github.com/alfreddobradi/actors/pkg/database"
	"github.com/alfreddobradi/actors/pkg/model"
	"github.com/alfreddobradi/actors/pkg/system"
	"github.com/google/uuid"
)

type KeeperActor struct {
	ID           uuid.UUID
	handle       repository.Repository
	sendCallback system.SenderFunc
}

func (k *KeeperActor) GetID() uuid.UUID {
	return k.ID
}

func (k *KeeperActor) GetKind() string {
	return "KeeperActor"
}

func (k *KeeperActor) HandleMessage(ctx context.Context, msg *system.Message) system.HandleError {
	switch m := msg.Payload.(type) {
	case GetGuildConfigRequest:
		slog.Info("received task to get guild config", "account_id", m.AccountID)
	}
	return nil
}

func (k *KeeperActor) Snapshot(ctx context.Context) (database.Snapshot, error) {
	// noop - this actor doesn't have any state to persist
	return database.Snapshot{}, nil
}

func (k *KeeperActor) RestoreFromSnapshot(ctx context.Context, snapshot database.Snapshot) error {
	// noop - this actor doesn't have any state to restore
	return nil
}

func (k *KeeperActor) Persist(ctx context.Context, db database.Store) error {
	return nil
}

func (k *KeeperActor) Restore(ctx context.Context, db database.Store) error {
	return nil
}

func (k *KeeperActor) Start(ctx context.Context) {
	// noop
}

func (k *KeeperActor) Stop(ctx context.Context) error {
	slog.Debug("Stopping keeper actor", "actor_id", k.GetID())
	return nil
}

func keeperActorFactory(ctx context.Context) system.Actor {
	interval := time.Second / game.TickRate

	slog.Debug("starting ticker actor", "interval_ms", interval.Milliseconds())

	handleAny := ctx.Value(model.ContextKeyDBHandle)
	if handleAny == nil {
		return nil
	}

	handle, ok := handleAny.(repository.Repository)
	if !ok {
		return nil
	}

	slog.Debug("starting keeper actor")

	return &KeeperActor{
		ID:           uuid.New(),
		handle:       handle,
		sendCallback: ctx.Value(model.ContextKeySenderFn).(system.SenderFunc),
	}
}

type QueryRequest interface {
	IsQuery()
}

type GetGuildConfigRequest struct {
	AccountID uuid.UUID
}

func (g GetGuildConfigRequest) IsQuery() {}

// func (k *KeeperActor) getGuildConfig(ctx context.Context, accountID uuid.UUID) (game.GuildConfig, error) {
// 	k.handle.
// }
