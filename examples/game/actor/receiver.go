package actor

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/alfreddobradi/actors/pkg/system"
	"github.com/google/uuid"
)

type ReceiveActor struct {
	ID uuid.UUID
}

func (h *ReceiveActor) GetID() uuid.UUID {
	return h.ID
}

func (h *ReceiveActor) GetKind() string {
	return "ReceiveActor"
}

func (h *ReceiveActor) HandleMessage(ctx context.Context, msg *system.Message) system.HandleError {
	slog.Info(
		"Received message",
		"kind", h.GetKind(),
		"actorID", h.GetID(),
		"senderID", msg.GetSender(),
		"messageID", msg.GetID(),
		"payload", fmt.Sprintf("%v", msg.GetBody()),
	)

	msg.Respond(h.GetID(), []byte("pong"))

	return nil
}

func (h *ReceiveActor) Start(ctx context.Context) {
	// noop - this actor only reacts to messages and doesn't have its own internal logic
}

func (h *ReceiveActor) Stop(ctx context.Context) error {
	slog.Info("Stopping receive actor", "actorID", h.GetID())
	return nil
}

func receiveActorFactory(ctx context.Context) system.Actor {
	return &ReceiveActor{
		ID: uuid.New(),
	}
}
