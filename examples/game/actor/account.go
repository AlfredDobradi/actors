package actor

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"math/rand"
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

type routeHandler func(ctx context.Context, msg *system.Message) system.HandleError

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
	handler := h.routeMessage(msg)

	if handler == nil {
		slog.Warn("Received message with unknown payload type", "actor_id", h.GetID(), "message_id", msg.GetID(), "payload_type", fmt.Sprintf("%T", msg.GetBody()))
		return NewErrInvalidMessage(fmt.Sprintf("%T", msg.GetBody()))
	}

	return handler(ctx, msg)
}

func (h *AccountActor) routeMessage(msg *system.Message) routeHandler {
	switch msg.GetBody().(type) {
	case Tick:
		return h.processTick
	case model.NewTavernRequest:
		return h.createTavern
	case model.GetCharacterRequest:
		return h.getCharacter
	case model.HireCharacterRequest:
		return h.hireCharacter
	default:
		return nil
	}
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

func (h *AccountActor) processTick(ctx context.Context, _ *system.Message) system.HandleError {
	spanID := telemetry.SpanIDFromContext(ctx)

	ctxLogger := slog.With("span_id", spanID)
	ctxLogger.Debug("Processing tick in account actor", "actor_id", h.GetID())

	h.Tavern.ProcessTick(ctx)
	return nil
}

func (h *AccountActor) replayTicks(ctx context.Context, since int64) {
	spanID := telemetry.SpanIDFromContext(ctx)

	ctxLogger := slog.With("span_id", spanID)

	sinceTime := time.Unix(since, 0)
	secondsSinceTime := time.Since(sinceTime).Seconds()
	ticksSinceTime := int(math.Floor(secondsSinceTime / float64(game.TickRate)))

	ctxLogger.Debug("Replaying ticks in account actor", "actor_id", h.GetID(), "since", sinceTime.Format(time.RFC3339), "ticks", ticksSinceTime)

	for i := 0; i < ticksSinceTime; i++ {
		h.processTick(ctx, &system.Message{})
	}

	ctxLogger.Debug("Finished replaying ticks in account actor", "actor_id", h.GetID(), "ticks_replayed", ticksSinceTime)
}

func (h *AccountActor) createTavern(ctx context.Context, msg *system.Message) system.HandleError {
	// tavern creation logic would go here, but for this example we'll just log the request and return an error since taverns aren't implemented
	slog.Info("Received request to create tavern", "actor_id", h.GetID(), "message_id", msg.GetID())

	if h.Tavern != nil {
		return ErrTavernExists{}
	}

	h.Tavern = game.NewTavern()

	if err := msg.Respond(h.ID, model.NewTavernResponse{OK: true}); err != nil {
		slog.Error("Failed to send tavern creation response", "error", err, "actor_id", h.GetID(), "message_id", msg.GetID())
		return NewErrResponseFailed(err)
	}

	return nil
}

// TODO check and remove gold
func (h *AccountActor) hireCharacter(ctx context.Context, msg *system.Message) system.HandleError {
	if h.Tavern == nil {
		return NewAccountError(fmt.Errorf("tavern not found for account"))
	}

	slog.Info("Received request to hire character", "actor_id", h.GetID(), "message_id", msg.GetID())

	firstNames := []string{"Arin", "Bel", "Cal", "Dain", "Eli"}
	lastNames := []string{"Strong", "Swift", "Brave", "Clever", "Bold"}

	name := fmt.Sprintf("%s %s", firstNames[rand.Intn(len(firstNames))], lastNames[rand.Intn(len(lastNames))])
	character := game.NewCharacter(name)

	h.Tavern.AddCharacter(&character)

	slog.Info("Hired new character", "actor_id", h.GetID(), "character_id", character.ID, "character_name", character.Name)

	response := model.HireCharacterResponse{
		OK:            true,
		CharacterID:   character.ID.String(),
		CharacterName: character.Name,
	}

	if err := msg.Respond(h.ID, response); err != nil {
		slog.Error("Failed to send hire character response", "error", err, "actor_id", h.GetID(), "message_id", msg.GetID())
		return NewErrResponseFailed(err)
	}

	return nil
}

func (h *AccountActor) getCharacter(ctx context.Context, msg *system.Message) system.HandleError {
	if h.Tavern == nil {
		return NewAccountError(fmt.Errorf("tavern not found for account"))
	}

	request, ok := msg.GetBody().(model.GetCharacterRequest)
	if !ok {
		return NewErrInvalidMessage(fmt.Sprintf("%T", msg.GetBody()))
	}

	id, err := uuid.Parse(request.ID)
	if err != nil {
		return NewAccountError(err)
	}

	slog.Info("Received request to get character", "actor_id", h.GetID(), "message_id", msg.GetID(), "character_id", id)

	character, exists := h.Tavern.GetCharacter(id)
	if !exists {
		return NewAccountError(fmt.Errorf("character with ID %s not found", id))
	}

	slog.Info("Found character", "actor_id", h.GetID(), "character_id", character.ID, "character_name", character.Name)

	response := model.GetCharacterResponse{
		Status:  "OK",
		Details: model.DetailsFromCharacter(character),
	}

	if err := msg.Respond(h.ID, response); err != nil {
		slog.Error("Failed to send get character response", "error", err, "actor_id", h.GetID(), "message_id", msg.GetID())
		return NewErrResponseFailed(err)
	}

	return nil
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
