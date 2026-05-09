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

func (h *AccountActor) Persist(ctx context.Context, db system.Persister) error {
	if db == nil {
		return fmt.Errorf("no database provided for persistence")
	}

	mapAccount := map[string]any{
		"id":         h.ID,
		"name":       h.Name,
		"characters": h.Characters.characters,
	}
	raw, err := json.Marshal(mapAccount)
	if err != nil {
		return err
	}

	return db.Set(ctx, fmt.Sprintf("actor:account:%s", h.ID), database.ToStringerable(string(raw)))
}

func (h *AccountActor) Restore(ctx context.Context, db system.Restorer) error {
	if db == nil {
		return fmt.Errorf("no database provided for restoration")
	}

	key := fmt.Sprintf("actor:account:%s", h.ID)
	data, ok := db.Get(ctx, key)
	if !ok {
		return fmt.Errorf("no data found in database for key %s", key)
	}

	var aux struct {
		ID         uuid.UUID                    `json:"id"`
		Name       string                       `json:"name"`
		Characters map[uuid.UUID]game.Character `json:"characters"`
	}
	if err := json.Unmarshal([]byte(data), &aux); err != nil {
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
	params, ok := ctx.Value(sysmodel.ContextKeyFactoryParams).(model.AccountActorParams)
	if !ok {
		params = model.AccountActorParams{ID: uuid.New(), Name: "default"}
	}

	if params.ID == uuid.Nil {
		params.ID = uuid.New()
	}

	if params.Name == "" {
		params.Name = "default"
	}

	return &AccountActor{
		ID:         params.ID,
		Name:       params.Name,
		Characters: newCharacterStore(),
	}
}
