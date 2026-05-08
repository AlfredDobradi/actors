package actor

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/alfreddobradi/actors/pkg/model"
	"github.com/alfreddobradi/actors/pkg/system"
	"github.com/google/uuid"
)

const (
	tickRate = 1 // ticks per second
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
	slog.Info("Received message", "actorID", h.GetID(), "messageID", msg.GetID(), "responseTo", msg.GetResponseTo(), "payload", fmt.Sprintf("%v", msg.GetBody()))
	return nil
}

func (h *TickerActor) Persist(ctx context.Context, db system.Persister) error {
	// noop - this actor doesn't have any state to persist
	return nil
}

func (h *TickerActor) Restore(ctx context.Context, db system.Restorer) error {
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
				slog.Info("Stopping ticker actor", "actorID", h.GetID())
				h.timer.Stop()
				return
			}
		}
	}()
}

func (h *TickerActor) Stop(ctx context.Context) error {
	slog.Info("Stopping ticker actor", "actorID", h.GetID())
	h.timer.Stop()
	return nil
}

func tickerActorFactory(ctx context.Context) system.Actor {
	interval := time.Second / tickRate

	slog.Debug("starting ticker actor", "interval_ms", interval.Milliseconds())

	return &TickerActor{
		ID:           uuid.New(),
		timer:        time.NewTicker(interval),
		sendCallback: ctx.Value(model.ContextKeySenderFn).(system.SenderFunc),
	}
}
