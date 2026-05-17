package actor

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"time"

	"github.com/alfreddobradi/actors/examples/game/game"
	"github.com/alfreddobradi/actors/examples/game/model"
	"github.com/alfreddobradi/actors/pkg/database"
	sysmodel "github.com/alfreddobradi/actors/pkg/model"
	"github.com/alfreddobradi/actors/pkg/system"
	"github.com/alfreddobradi/actors/pkg/telemetry"
	"github.com/google/uuid"
)

type AccountActor struct {
	ID     uuid.UUID
	Name   string
	Tavern *game.Tavern
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
		h.ProcessTick(ctx)
	case model.StartActionRequest:
		// h.startAction(ctx, payload.CharacterID, payload.Action)
	case model.StopActionRequest:
		// h.stopAction(ctx, payload.CharacterID)
	case model.CreateCharacterRequest:
		// h.addCharacter(ctx, payload)
	case model.GetCharacterRequest:
		// character := h.getCharacter(ctx, payload)
		// if character != nil {
		// 	slog.Info("Character found", "actor_id", h.GetID(), "character_id", character.ID, "character_name", character.Name, "character_level", character.Level, "character_experience", character.Experience, "character_status", character.Status)
		// 	if err := msg.Respond(h.GetID(), character); err != nil {
		// 		slog.Warn("Failed to respond to request", "error", err)
		// 	}
		// } else {
		// 	slog.Debug("Character not found", "actor_id", h.GetID(), "character_name", payload.Name)
		// }
	default:
		slog.Warn("Received message with unknown payload type", "actor_id", h.GetID(), "message_id", msg.GetID(), "payload_type", fmt.Sprintf("%T", payload))
	}
	return nil
}

func (h *AccountActor) Snapshot(ctx context.Context) (database.Snapshot, error) {
	mapAccount := map[string]any{
		"id":         h.ID,
		"name":       h.Name,
		"characters": h.Tavern.Characters(),
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
	h.Tavern = game.NewTavern()
	for _, character := range aux.Characters {
		h.Tavern.AddCharacter(&character)
	}

	return nil
}

func (h *AccountActor) Start(ctx context.Context) {
	// noop - this actor only reacts to messages and doesn't have its own internal logic
}

func (h *AccountActor) Stop(ctx context.Context) error {
	slog.Debug("Stopping account actor", "actor_id", h.GetID())
	return nil
}

func (h *AccountActor) ProcessTick(ctx context.Context) {
	spanID := telemetry.SpanIDFromContext(ctx)

	ctxLogger := slog.With("span_id", spanID)
	ctxLogger.Debug("Processing tick in account actor", "actor_id", h.GetID())

	h.Tavern.ProcessTick(ctx)
}

func (h *AccountActor) ReplayTicks(ctx context.Context, since int64) {
	spanID := telemetry.SpanIDFromContext(ctx)

	ctxLogger := slog.With("span_id", spanID)

	sinceTime := time.Unix(since, 0)
	secondsSinceTime := time.Since(sinceTime).Seconds()
	ticksSinceTime := int(math.Floor(secondsSinceTime / float64(game.TickRate)))

	ctxLogger.Debug("Replaying ticks in account actor", "actor_id", h.GetID(), "since", sinceTime.Format(time.RFC3339), "ticks", ticksSinceTime)

	for i := 0; i < ticksSinceTime; i++ {
		h.ProcessTick(ctx)
	}

	ctxLogger.Debug("Finished replaying ticks in account actor", "actor_id", h.GetID(), "ticks_replayed", ticksSinceTime)
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
		ID:     accountParams.ID,
		Name:   accountParams.Name,
		Tavern: nil,
	}
}
