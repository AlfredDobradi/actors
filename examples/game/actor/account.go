package actor

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"sync/atomic"
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
	Gold   *atomic.Uint64
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
	raw, err := json.Marshal(h)
	if err != nil {
		return database.Snapshot{}, err
	}

	return database.NewSnapshot(raw), nil
}

func (h *AccountActor) RestoreFromSnapshot(ctx context.Context, snapshot database.Snapshot) error {
	if snapshot.Data == nil {
		return fmt.Errorf("no snapshot data provided for restoration")
	}

	var aux AccountActor
	if err := json.Unmarshal(snapshot.Data, &aux); err != nil {
		return err
	}

	h.ID = aux.ID
	h.Name = aux.Name
	h.Tavern = aux.Tavern
	if aux.Gold == nil {
		h.Gold = &atomic.Uint64{}
	} else {
		h.Gold = aux.Gold
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

	if accountParams.ID == uuid.Nil {
		accountParams.ID = uuid.New()
	}

	startingMoney := &atomic.Uint64{}
	startingMoney.Store(3000)

	a := &AccountActor{
		ID:     accountParams.ID,
		Name:   accountParams.Name,
		Tavern: nil,
		Gold:   startingMoney,
	}

	return a
}

func (a *AccountActor) MarshalJSON() ([]byte, error) {
	gold := uint64(0)
	if a.Gold != nil {
		gold = a.Gold.Load()
	}

	raw := map[string]any{
		"id":         a.ID,
		"name":       a.Name,
		"characters": a.Tavern.Characters(),
		"gold":       gold,
	}
	return json.Marshal(raw)
}

func (a *AccountActor) UnmarshalJSON(data []byte) error {
	var aux struct {
		ID         uuid.UUID                    `json:"id"`
		Name       string                       `json:"name"`
		Characters map[uuid.UUID]game.Character `json:"characters"`
		Gold       uint64                       `json:"gold"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	a.ID = aux.ID
	a.Name = aux.Name
	a.Tavern = game.NewTavern()
	for _, character := range aux.Characters {
		a.Tavern.AddCharacter(&character)
	}
	a.Gold = &atomic.Uint64{}
	a.Gold.Store(aux.Gold)

	return nil
}
