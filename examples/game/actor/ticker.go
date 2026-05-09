package actor

import (
	"context"
	"log/slog"
	"time"

	"github.com/alfreddobradi/actors/examples/game/game"
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

func (h *TickerActor) GetID() uuid.UUID {
	return h.ID
}

func (h *TickerActor) GetKind() string {
	return "TickerActor"
}

func (h *TickerActor) HandleMessage(ctx context.Context, msg *system.Message) system.HandleError {
	// this actor should never receive any messages
	return nil
}

func (h *TickerActor) Snapshot(ctx context.Context) (database.Snapshot, error) {
	// noop - this actor doesn't have any state to persist
	return database.Snapshot{}, nil
}

func (h *TickerActor) RestoreFromSnapshot(ctx context.Context, snapshot database.Snapshot) error {
	// noop - this actor doesn't have any state to restore
	return nil
}

func (h *TickerActor) tickCallback(ctx context.Context) error {
	spanID := uuid.New()
	sctx := context.WithValue(ctx, model.ContextKeySpanID, spanID)

	slog.Debug("Sending tick message", "span_id", spanID, "actorID", h.GetID())

	_, err := h.sendCallback(
		sctx,
		false, // we don't expect response to ticks
		h.GetID(),
		system.Recipient{Kind: system.RecipientKindTopic, Subject: "ticks"},
		NewTick(),
	)
	if err != nil {
		return err
	}

	return nil
}

func (h *TickerActor) Start(ctx context.Context) {
	go func() {
		for {
			select {
			case <-h.timer.C:
				if err := h.tickCallback(ctx); err != nil {
					slog.Error("Failed to send tick message", "actorID", h.GetID(), "error", err)
				}
			case <-ctx.Done():
				h.timer.Stop()
				return
			}
		}
	}()
}

func (h *TickerActor) Stop(ctx context.Context) error {
	slog.Debug("Stopping ticker actor", "actorID", h.GetID())
	h.timer.Stop()
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
