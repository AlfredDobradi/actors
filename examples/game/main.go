package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alfreddobradi/actors/pkg/system"
	"github.com/google/uuid"
)

type TickerActor struct {
	ID    uuid.UUID
	timer *time.Ticker
}

func (h *TickerActor) GetID() uuid.UUID {
	return h.ID
}

func (h *TickerActor) GetKind() string {
	return "TickerActor"
}

func (h *TickerActor) HandleMessage(ctx context.Context, msg system.Message) system.HandleError {
	slog.Info("Received message", "actorID", h.GetID(), "messageID", msg.ID, "payload", string(msg.Payload))
	return nil
}

func (h *TickerActor) Start(ctx context.Context) {
	go func() {
		for {
			select {
			case <-h.timer.C:
				slog.Info("Ticker", "actorID", h.GetID())
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
	return &TickerActor{
		ID:    uuid.New(),
		timer: time.NewTicker(1 * time.Second),
	}
}

func main() {
	slog.SetDefault(
		slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})),
	)

	registry := system.NewRegistry()
	registry.RegisterFactory("TickerActor", tickerActorFactory)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sys := system.NewSystem(nil, registry)
	handler, err := sys.Spawn(
		ctx,
		"TickerActor",
	)
	if err != nil {
		slog.Error("Failed to spawn actor", "error", err)
		return
	}

	// Wait for interrupt signal (Ctrl+C)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	slog.Info("Interrupt received, stopping actor")
	handler.Stop()

	handler.WaitForTermination()
}
