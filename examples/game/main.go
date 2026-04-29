package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alfreddobradi/actors/pkg/system"
	"github.com/google/uuid"
)

var receiveActorId = uuid.MustParse("123e4567-e89b-12d3-a456-426614174001")

type TickerActor struct {
	ID     uuid.UUID
	timer  *time.Ticker
	sender system.MessageConsumer
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
				h.sender.Inbox() <- &TestRequest{
					id:     uuid.New(),
					body:   "Tick",
					topic:  fmt.Sprintf("actor:%s", receiveActorId.String()),
					sender: h.GetID(),
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
	return &TickerActor{
		ID:     uuid.New(),
		timer:  time.NewTicker(1 * time.Second),
		sender: ctx.Value(system.ContextKeySender).(system.MessageConsumer),
	}
}

type ReceiveActor struct {
	ID    uuid.UUID
	timer *time.Ticker
}

func (h *ReceiveActor) GetID() uuid.UUID {
	return h.ID
}

func (h *ReceiveActor) GetKind() string {
	return "ReceiveActor"
}

func (h *ReceiveActor) HandleMessage(ctx context.Context, msg system.Message) system.HandleError {
	slog.Info("Received message", "actorID", h.GetID(), "messageID", msg.ID, "payload", string(msg.Payload))

	return nil
}

func (h *ReceiveActor) Start(ctx context.Context) {
	// noop - this actor only reacts to messages and doesn't have its own internal logic
}

func (h *ReceiveActor) Stop(ctx context.Context) error {
	slog.Info("Stopping receive actor", "actorID", h.GetID())
	h.timer.Stop()
	return nil
}

func receiveActorFactory(ctx context.Context) system.Actor {
	return &ReceiveActor{
		ID:    receiveActorId,
		timer: time.NewTicker(1 * time.Second),
	}
}

type TestRequest struct {
	id     uuid.UUID
	body   string
	topic  string
	sender uuid.UUID
}

func (r *TestRequest) GetSender() uuid.UUID {
	return r.sender
}

func (r *TestRequest) GetID() uuid.UUID {
	return r.id
}

func (r *TestRequest) GetBody() []byte {
	return []byte(r.body)
}

func (r *TestRequest) GetTopic() string {
	return r.topic
}

func main() {
	slog.SetDefault(
		slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})),
	)

	registry := system.NewRegistry()
	registry.RegisterFactory("TickerActor", tickerActorFactory)
	registry.RegisterFactory("ReceiverActor", receiveActorFactory)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sys := system.NewSystem(registry)
	handlerTicker, err := sys.Spawn(ctx, "TickerActor")
	if err != nil {
		slog.Error("Failed to spawn actor", "error", err)
		return
	}

	if _, err := sys.Spawn(ctx, "ReceiverActor"); err != nil {
		slog.Error("Failed to spawn actor", "error", err)
		return
	}

	msg := &TestRequest{
		id:     uuid.New(),
		body:   "Hello, Actor!",
		topic:  handlerTicker.GetAddress(),
		sender: uuid.New(),
	}

	sys.Send(msg)

	// Wait for interrupt signal (Ctrl+C)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	slog.Info("Interrupt received, stopping actor")
	handlerTicker.Stop()

	handlerTicker.WaitForTermination()
}
