package actor

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/alfreddobradi/actors/examples/game/game"
	"github.com/alfreddobradi/actors/examples/game/model"
	"github.com/alfreddobradi/actors/pkg/system"
	"github.com/google/uuid"
)

type characterStore struct {
	mx         *sync.Mutex
	characters map[uuid.UUID]game.Character
}

type AccountActor struct {
	ID         uuid.UUID
	Name       string
	Characters *characterStore
}

func (h *AccountActor) GetID() uuid.UUID {
	return h.ID
}

func (h *AccountActor) GetKind() string {
	return "AccountActor"
}

func (h *AccountActor) HandleMessage(ctx context.Context, msg *system.Message) system.HandleError {
	switch payload := msg.GetBody().(type) {
	case Tick:
		// h.processTick(ctx)
	case model.StartActionRequest:
		// h.startAction(ctx, payload.CharacterID, payload.Action)
	case model.StopActionRequest:
		// h.stopAction(ctx, payload.CharacterID)
	case model.CreateCharacterRequest:
		// h.addCharacter(ctx, payload)
	case model.GetCharacterRequest:
		// character := h.getCharacter(ctx, payload)
		// if character != nil {
		// 	slog.Info("Character found", "actorID", h.GetID(), "characterID", character.ID, "characterName", character.Name, "characterLevel", character.Level, "characterExperience", character.Experience, "characterStatus", character.Status)
		// 	if err := msg.Respond(h.GetID(), character); err != nil {
		// 		slog.Warn("Failed to respond to request", "error", err)
		// 	}
		// } else {
		// 	slog.Debug("Character not found", "actorID", h.GetID(), "characterName", payload.Name)
		// }
	default:
		slog.Warn("Received message with unknown payload type", "actorID", h.GetID(), "messageID", msg.GetID(), "payloadType", fmt.Sprintf("%T", payload))
	}
	return nil
}

func (h *AccountActor) Start(ctx context.Context) {
	// noop - this actor only reacts to messages and doesn't have its own internal logic
}

func (h *AccountActor) Stop(ctx context.Context) error {
	slog.Info("Stopping account actor", "actorID", h.GetID())
	return nil
}

func newCharacterStore() *characterStore {
	return &characterStore{
		mx:         &sync.Mutex{},
		characters: make(map[uuid.UUID]game.Character),
	}
}

type AccountActorParams struct {
	Name string
}

func accountActorFactory(ctx context.Context) system.Actor {
	params, ok := ctx.Value(system.ContextKeyFactoryParams).(AccountActorParams)
	if !ok {
		params = AccountActorParams{Name: "default"}
	}

	return &AccountActor{
		ID:         uuid.New(),
		Name:       params.Name,
		Characters: newCharacterStore(),
	}
}
