package actor

import (
	"context"
	"log/slog"
	"time"

	"github.com/alfreddobradi/actors/cmd/game/game"
	"github.com/alfreddobradi/actors/pkg/database"
	"github.com/alfreddobradi/actors/pkg/model"
	"github.com/alfreddobradi/actors/pkg/system"
	"github.com/google/uuid"
)

type TickerActor struct {
	ID           uuid.UUID
	timer        *time.Ticker
	sendCallback system.SenderFunc
}

type Tick struct {
	Timestamp time.Time
}

func NewTick() Tick {
	return Tick{
		Timestamp: time.Now(),
	}
}

func (t *TickerActor) GetID() uuid.UUID {
	return t.ID
}

func (t *TickerActor) GetKind() string {
	return "TickerActor"
}

func (t *TickerActor) HandleMessage(ctx context.Context, msg *system.Message) system.HandleError {
	// this actor should never receive any messages
	return nil
}

func (t *TickerActor) Snapshot(ctx context.Context) (database.Snapshot, error) {
	// noop - this actor doesn't have any state to persist
	return database.Snapshot{}, nil
}

func (t *TickerActor) RestoreFromSnapshot(ctx context.Context, snapshot database.Snapshot) error {
	// noop - this actor doesn't have any state to restore
	return nil
}

func (t *TickerActor) Persist(ctx context.Context, db database.Store) error {
	return nil
}

func (t *TickerActor) Restore(ctx context.Context, db database.Store) error {
	return nil
}

func (t *TickerActor) tickCallback(ctx context.Context) error {
	spanID := uuid.New()
	sctx := context.WithValue(ctx, model.ContextKeySpanID, spanID)

	slog.Debug("Sending tick message", "span_id", spanID, "actorID", t.GetID())

	_, err := t.sendCallback(
		sctx,
		false, // we don't expect response to ticks
		t.GetID(),
		system.Recipient{Kind: system.RecipientKindTopic, Subject: "ticks"},
		NewTick(),
	)
	if err != nil {
		return err
	}

	return nil
}

func (t *TickerActor) Start(ctx context.Context) {
	go func() {
		for {
			select {
			case <-t.timer.C:
				if err := t.tickCallback(ctx); err != nil {
					slog.Error("Failed to send tick message", "actorID", t.GetID(), "error", err)
				}
			case <-ctx.Done():
				t.timer.Stop()
				return
			}
		}
	}()
}

func (t *TickerActor) Stop(ctx context.Context) error {
	slog.Debug("Stopping ticker actor", "actorID", t.GetID())
	t.timer.Stop()
	return nil
}

func tickerActorFactory(ctx context.Context) system.Actor {
	interval := time.Second / game.TickRate

	slog.Debug("starting ticker actor", "interval_ms", interval.Milliseconds())

	return &TickerActor{
		ID:           uuid.New(),
		timer:        time.NewTicker(interval),
		sendCallback: ctx.Value(model.ContextKeySenderFn).(system.SenderFunc),
	}
}
