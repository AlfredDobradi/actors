package actor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"

	"github.com/alfreddobradi/actors/examples/game/game"
	"github.com/alfreddobradi/actors/examples/game/model"
	"github.com/alfreddobradi/actors/pkg/database"
	sysmodel "github.com/alfreddobradi/actors/pkg/model"
	"github.com/alfreddobradi/actors/pkg/system"
	"github.com/google/uuid"
)

type characterStore struct {
	mx         *sync.Mutex
	characters map[uuid.UUID]game.Character
}

func (s *characterStore) encode() ([]byte, error) {
	s.mx.Lock()
	defer s.mx.Unlock()

	// Serialize the character store to a byte slice
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	if err := enc.Encode(s.characters); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
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

func (h *AccountActor) UnmarshalJSON(data []byte) error {
	var aux struct {
		ID         uuid.UUID                    `json:"id"`
		Name       string                       `json:"name"`
		Characters map[uuid.UUID]game.Character `json:"characters"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	h.ID = aux.ID
	h.Name = aux.Name
	h.Characters = &characterStore{
		mx:         &sync.Mutex{},
		characters: aux.Characters,
	}

	return nil
}

func (h *AccountActor) Snapshot(ctx context.Context) (database.Snapshot, error) {
	mapAccount := map[string]any{
		"id":         h.ID,
		"name":       h.Name,
		"characters": h.Characters.characters,
	}
	raw, err := json.Marshal(mapAccount)
	if err != nil {
		return database.Snapshot{}, err
	}

	return database.NewSnapshot(raw), nil
}

func (h *AccountActor) RestoreFromSnapshot(ctx context.Context, snapshot database.Snapshot) error {
	if snapshot.Data == nil {
		return fmt.Errorf("no snapshot data provided for restoration")
	}

	var aux struct {
		ID         uuid.UUID                    `json:"id"`
		Name       string                       `json:"name"`
		Characters map[uuid.UUID]game.Character `json:"characters"`
	}
	if err := json.Unmarshal(snapshot.Data, &aux); err != nil {
		return err
	}

	h.ID = aux.ID
	h.Name = aux.Name
	h.Characters = &characterStore{
		mx:         &sync.Mutex{},
		characters: aux.Characters,
	}

	return nil
}

func (h *AccountActor) Start(ctx context.Context) {
	// noop - this actor only reacts to messages and doesn't have its own internal logic
}

func (h *AccountActor) Stop(ctx context.Context) error {
	slog.Debug("Stopping account actor", "actorID", h.GetID())
	return nil
}

func newCharacterStore() *characterStore {
	return &characterStore{
		mx:         &sync.Mutex{},
		characters: make(map[uuid.UUID]game.Character),
	}
}

func accountActorFactory(ctx context.Context) system.Actor {
	accountParams := model.AccountActorParams{
		ID:   uuid.New(),
		Name: "default",
	}

	params := ctx.Value(sysmodel.ContextKeyFactoryParams)
	if params != nil {
		switch p := params.(type) {
		case model.AccountActorParams:
			accountParams = p
		case sysmodel.IDParam:
			accountParams.ID = p.GetID()
		default:
			slog.Warn("Received unexpected factory params type, using default params", "expectedType", fmt.Sprintf("%T", model.AccountActorParams{}), "actualType", fmt.Sprintf("%T", params))
			params = model.AccountActorParams{ID: uuid.New(), Name: "default"}
		}
	}

	return &AccountActor{
		ID:         accountParams.ID,
		Name:       accountParams.Name,
		Characters: newCharacterStore(),
	}
}
