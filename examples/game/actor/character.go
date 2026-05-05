package actor

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/alfreddobradi/actors/examples/game/game"
	"github.com/alfreddobradi/actors/examples/game/model"
	"github.com/alfreddobradi/actors/examples/game/telemetry"
	"github.com/alfreddobradi/actors/pkg/system"
	"github.com/google/uuid"
)

type CharacterStore struct {
	mx         *sync.Mutex
	characters map[uuid.UUID]game.Character
}

func (h *CharacterStore) GetID() uuid.UUID {
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte("character-store"))
}

func (h *CharacterStore) GetKind() string {
	return "CharacterStore"
}

func (h *CharacterStore) HandleMessage(ctx context.Context, msg *system.Message) system.HandleError {
	switch payload := msg.GetBody().(type) {
	case Tick:
		h.processTick(ctx)
	case model.StartActionRequest:
		h.startAction(ctx, payload.CharacterID, payload.Action)
	case model.StopActionRequest:
		h.stopAction(ctx, payload.CharacterID)
	case model.CreateCharacterRequest:
		h.addCharacter(ctx, payload)
	case model.GetCharacterRequest:
		character := h.getCharacter(ctx, payload)
		if character != nil {
			slog.Info("Character found", "actorID", h.GetID(), "characterID", character.ID, "characterName", character.Name, "characterLevel", character.Level, "characterExperience", character.Experience, "characterStatus", character.Status)
			if err := msg.Respond(h.GetID(), character); err != nil {
				slog.Warn("Failed to respond to request", "error", err)
			}
		} else {
			slog.Debug("Character not found", "actorID", h.GetID(), "characterName", payload.Name)
		}
	default:
		slog.Warn("Received message with unknown payload type", "actorID", h.GetID(), "messageID", msg.GetID(), "payloadType", fmt.Sprintf("%T", payload))
	}
	return nil
}

func (h *CharacterStore) Start(ctx context.Context) {
	// noop - this actor only reacts to messages and doesn't have its own internal logic
}

func (h *CharacterStore) Stop(ctx context.Context) error {
	slog.Info("Stopping character store", "actorID", h.GetID())
	return nil
}

func characterStoreFactory(ctx context.Context) system.Actor {
	return &CharacterStore{
		mx:         &sync.Mutex{},
		characters: make(map[uuid.UUID]game.Character),
	}
}

func (h *CharacterStore) processTick(ctx context.Context) {
	spanID := telemetry.SpanIDFromContext(ctx)

	ctxLogger := slog.With("span_id", spanID)
	ctxLogger.Debug("Processing tick in character store", "actorID", h.GetID())

	h.mx.Lock()
	defer h.mx.Unlock()
	for id, character := range h.characters {
		character.ProcessTick(ctx)
		h.characters[id] = character
		ctxLogger.Debug("Processed tick for character", "actorID", h.GetID(), "characterID", character.ID, "characterName", character.Name, "characterLevel", character.Level, "characterExperience", character.Experience, "characterStatus", character.Status)
	}
}

func (h *CharacterStore) addCharacter(ctx context.Context, character model.CreateCharacterRequest) {
	spanID := telemetry.SpanIDFromContext(ctx)

	ctxLogger := slog.With("span_id", spanID)
	ctxLogger.Info("Adding character to store", "actorID", h.GetID(), "characterName", character.Name)

	ch := game.NewCharacter(character.Name)

	h.mx.Lock()
	h.characters[ch.ID] = ch
	h.mx.Unlock()

	ctxLogger.Info("Character added to store", "actorID", h.GetID(), "characterID", ch.ID)
}

func (h *CharacterStore) getCharacter(ctx context.Context, request model.GetCharacterRequest) *game.Character {
	spanID := telemetry.SpanIDFromContext(ctx)

	ctxLogger := slog.With("span_id", spanID)
	ctxLogger.Info("Getting character from store", "actorID", h.GetID(), "characterName", request.Name)

	h.mx.Lock()
	defer h.mx.Unlock()

	for _, character := range h.characters {
		if character.Name == request.Name {
			return &character
		}
	}

	return nil
}

func (h *CharacterStore) startAction(ctx context.Context, character uuid.UUID, action game.Action) {
	spanID := telemetry.SpanIDFromContext(ctx)
	ctxLogger := slog.With("span_id", spanID)
	ctxLogger.Info("Sending start action signal", "actorID", h.GetID(), "characterID", character, "actionType", fmt.Sprintf("%T", action))

	h.mx.Lock()
	defer h.mx.Unlock()

	ch, exists := h.characters[character]
	if !exists {
		ctxLogger.Warn("Character not found in store, cannot start action", "characterID", character)
		return
	}

	ch.StartAction(ctx, action)
	h.characters[character] = ch
}

func (h *CharacterStore) stopAction(ctx context.Context, character uuid.UUID) {
	spanID := telemetry.SpanIDFromContext(ctx)
	ctxLogger := slog.With("span_id", spanID)
	ctxLogger.Info("Stopping action for character", "actorID", h.GetID(), "characterID", character)

	h.mx.Lock()
	defer h.mx.Unlock()

	ch, exists := h.characters[character]
	if !exists {
		ctxLogger.Warn("Character not found in store, cannot stop action", "characterID", character)
		return
	}

	ch.StopAction(ctx)
	h.characters[character] = ch
}
