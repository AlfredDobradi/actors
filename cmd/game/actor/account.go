package actor

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/alfreddobradi/actors/cmd/game/game"
	"github.com/alfreddobradi/actors/cmd/game/model"
	"github.com/alfreddobradi/actors/pkg/database"
	sysmodel "github.com/alfreddobradi/actors/pkg/model"
	"github.com/alfreddobradi/actors/pkg/system"
	"github.com/alfreddobradi/actors/pkg/telemetry"
	"github.com/google/uuid"
)

type routeHandler func(ctx context.Context, msg *system.Message) system.HandleError

type AccountActor struct {
	mx *sync.Mutex

	ID     uuid.UUID
	Name   string
	Tavern *game.Tavern
	Gold   *atomic.Int64
}

func (a *AccountActor) GetID() uuid.UUID {
	return a.ID
}

func (a *AccountActor) GetKind() string {
	return "AccountActor"
}

func (a *AccountActor) HandleMessage(ctx context.Context, msg *system.Message) system.HandleError {
	handler := a.routeMessage(msg)

	if handler == nil {
		slog.Warn("Received message with unknown payload type", "actor_id", a.GetID(), "message_id", msg.GetID(), "payload_type", fmt.Sprintf("%T", msg.GetBody()))
		return NewErrInvalidMessage(fmt.Sprintf("%T", msg.GetBody()))
	}

	return handler(ctx, msg)
}

func (a *AccountActor) routeMessage(msg *system.Message) routeHandler {
	switch msg.GetBody().(type) {
	case Tick:
		return a.processTick
	case model.NewTavernRequest:
		return a.createTavern
	case model.GetCharactersRequest:
		return a.getCharacters
	case model.GetCharacterRequest:
		return a.getCharacter
	case model.HireCharacterRequest:
		return a.hireCharacter
	default:
		return nil
	}
}

func (a *AccountActor) Snapshot(ctx context.Context) (database.Snapshot, error) {
	raw, err := json.Marshal(a)
	if err != nil {
		return database.Snapshot{}, err
	}

	return database.NewSnapshot(raw), nil
}

func (a *AccountActor) restore(ctx context.Context, snapshot database.Snapshot) error {
	if snapshot.Data == nil {
		return fmt.Errorf("no snapshot data provided for restoration")
	}

	var aux AccountActor
	if err := json.Unmarshal(snapshot.Data, &aux); err != nil {
		return err
	}

	a.ID = aux.ID
	a.Name = aux.Name
	a.Tavern = aux.Tavern
	if aux.Gold == nil {
		a.Gold = &atomic.Int64{}
	} else {
		a.Gold = aux.Gold
	}

	return nil
}

func (a *AccountActor) RestoreFromSnapshot(ctx context.Context, snapshot database.Snapshot) error {
	if err := a.restore(ctx, snapshot); err != nil {
		return err
	}

	return a.replayTicks(ctx, snapshot.Timestamp)
}

func (a *AccountActor) Start(ctx context.Context) {
	// noop - this actor only reacts to messages and doesn't have its own internal logic
}

func (a *AccountActor) Stop(ctx context.Context) error {
	slog.Debug("Stopping account actor", "actor_id", a.GetID())
	return nil
}

func (a *AccountActor) processTick(ctx context.Context, _ *system.Message) system.HandleError {
	spanID := telemetry.SpanIDFromContext(ctx)

	ctxLogger := slog.With("span_id", spanID)
	ctxLogger.Debug("Processing tick in account actor", "actor_id", a.GetID())

	if a.Tavern != nil {
		a.Tavern.ProcessTick(ctx)
	}
	return nil
}

func (a *AccountActor) replayTicks(ctx context.Context, since int64) error {
	spanID := telemetry.SpanIDFromContext(ctx)

	ctxLogger := slog.With("span_id", spanID)

	sinceTime := time.Unix(since, 0)
	secondsSinceTime := time.Since(sinceTime).Seconds()
	ticksSinceTime := int(math.Floor(secondsSinceTime / float64(game.TickRate)))

	ctxLogger.Debug("Replaying ticks in account actor", "actor_id", a.GetID(), "since", sinceTime.Format(time.RFC3339), "ticks", ticksSinceTime)

	tempActor := &AccountActor{}
	snapshot, err := a.Snapshot(ctx)
	if err != nil {
		return err
	}

	if err := tempActor.restore(ctx, database.Snapshot{Data: snapshot.Data}); err != nil {
		ctxLogger.Error("Failed to restore account actor from snapshot for replay", "error", err, "actor_id", a.GetID())
		return err
	}

	if errs := tempActor.Tavern.ReplayTicks(ctx, ticksSinceTime); errs != nil {
		ctxLogger.Error("Failed to replay ticks in tavern during account actor replay", "error", err, "actor_id", a.GetID())
		return err
	}

	if a.Gold == nil {
		a.Gold = &atomic.Int64{}
	}
	// if we replayed in the temporary actor we copy the relevant state
	a.Gold.Store(tempActor.Gold.Load())
	a.Tavern = tempActor.Tavern

	ctxLogger.Debug("Finished replaying ticks in account actor", "actor_id", a.GetID(), "ticks_replayed", ticksSinceTime)
	return nil
}

func (a *AccountActor) createTavern(ctx context.Context, msg *system.Message) system.HandleError {
	// tavern creation logic would go here, but for this example we'll just log the request and return an error since taverns aren't implemented
	slog.Info("Received request to create tavern", "actor_id", a.GetID(), "message_id", msg.GetID())

	request, ok := msg.GetBody().(model.NewTavernRequest)
	if !ok {
		return NewErrInvalidMessage(fmt.Sprintf("%T", msg.GetBody()))
	}

	slog.Info("Creating tavern", "actor_id", a.GetID(), "message_id", msg.GetID(), "tavern_name", request.Name)

	if a.Tavern != nil {
		return ErrTavernExists{}
	}

	if request.Name == "" {
		msg.Respond(a.ID, model.NewTavernResponse{OK: false, Error: "tavern name cannot be empty"}) //nolint:errcheck
		return NewAccountError(fmt.Errorf("tavern name cannot be empty"))
	}

	a.mx.Lock()
	a.Tavern = game.NewTavern(request.Name)
	a.mx.Unlock()

	if err := msg.Respond(a.ID, model.NewTavernResponse{Name: a.Tavern.Name(), OK: true}); err != nil {
		slog.Error("Failed to send tavern creation response", "error", err, "actor_id", a.GetID(), "message_id", msg.GetID())
		return NewErrResponseFailed(err)
	}

	return nil
}

func (a *AccountActor) hireCharacter(ctx context.Context, msg *system.Message) system.HandleError {
	if a.Tavern == nil {
		msg.Respond(a.ID, model.HireCharacterResponse{OK: false, Error: "tavern not found for account"}) //nolint:errcheck

		return NewAccountError(fmt.Errorf("tavern not found for account"))
	}

	slog.Info("Received request to hire character", "actor_id", a.GetID(), "message_id", msg.GetID())

	// check whether account has enough gold
	cost := game.HeroPriceMultiplier(len(a.Tavern.Characters())) * 1000

	if cost > a.Gold.Load() {
		msg.Respond(a.ID, model.HireCharacterResponse{OK: false, Error: fmt.Sprintf("not enough gold: have %d, need %d", a.Gold.Load(), cost)}) //nolint:errcheck
		return NewAccountError(fmt.Errorf("not enough gold to hire character: have %d, need %d", a.Gold.Load(), cost))
	} else {
		a.Gold.Add(-cost)
	}

	character := game.GenerateRandomCharacter()

	a.Tavern.AddCharacter(&character)

	slog.Info("Hired new character", "actor_id", a.GetID(), "character_id", character.ID, "character_name", character.Name)

	response := model.HireCharacterResponse{
		OK:            true,
		CharacterID:   character.ID.String(),
		CharacterName: character.Name,
	}

	if err := msg.Respond(a.ID, response); err != nil {
		slog.Error("Failed to send hire character response", "error", err, "actor_id", a.GetID(), "message_id", msg.GetID())
		return NewErrResponseFailed(err)
	}

	return nil
}

func (a *AccountActor) getCharacter(ctx context.Context, msg *system.Message) system.HandleError {
	if a.Tavern == nil {
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

	slog.Info("Received request to get character", "actor_id", a.GetID(), "message_id", msg.GetID(), "character_id", id)

	character, exists := a.Tavern.GetCharacter(id)
	if !exists {
		return NewAccountError(fmt.Errorf("character with ID %s not found", id))
	}

	slog.Info("Found character", "actor_id", a.GetID(), "character_id", character.ID, "character_name", character.Name)

	response := model.GetCharacterResponse{
		Status:  "OK",
		Details: model.DetailsFromCharacter(character),
	}

	if err := msg.Respond(a.ID, response); err != nil {
		slog.Error("Failed to send get character response", "error", err, "actor_id", a.GetID(), "message_id", msg.GetID())
		return NewErrResponseFailed(err)
	}

	return nil
}

func (a *AccountActor) getCharacters(ctx context.Context, msg *system.Message) system.HandleError {
	if a.Tavern == nil {
		return NewAccountError(fmt.Errorf("tavern not found for account"))
	}

	slog.Info("Received request to get characters", "actor_id", a.GetID(), "message_id", msg.GetID())

	characters := a.Tavern.Characters()
	if len(characters) == 0 {
		return NewAccountError(fmt.Errorf("no characters found"))
	}

	slog.Info("Found characters", "actor_id", a.GetID(), "character_count", len(characters))

	details := make([]model.CharacterDetails, 0, len(characters))
	for _, character := range characters {
		details = append(details, model.DetailsFromCharacter(character))
	}

	response := model.GetCharactersResponse{
		Status:  "OK",
		Details: details,
	}

	if err := msg.Respond(a.ID, response); err != nil {
		slog.Error("Failed to send get characters response", "error", err, "actor_id", a.GetID(), "message_id", msg.GetID())
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
			accountParams = model.AccountActorParams{ID: uuid.New(), Name: "default"}
		}
	}

	if accountParams.ID == uuid.Nil {
		accountParams.ID = uuid.New()
	}

	startingMoney := &atomic.Int64{}
	startingMoney.Store(3000)

	a := &AccountActor{
		mx:     &sync.Mutex{},
		ID:     accountParams.ID,
		Name:   accountParams.Name,
		Tavern: nil,
		Gold:   startingMoney,
	}

	return a
}

func (a *AccountActor) MarshalJSON() ([]byte, error) {
	gold := int64(0)
	if a.Gold != nil {
		gold = a.Gold.Load()
	}

	raw := map[string]any{
		"id":     a.ID,
		"name":   a.Name,
		"gold":   gold,
		"tavern": a.Tavern,
	}

	return json.Marshal(raw)
}

func (a *AccountActor) UnmarshalJSON(data []byte) error {
	var aux struct {
		ID     uuid.UUID    `json:"id"`
		Name   string       `json:"name"`
		Gold   int64        `json:"gold"`
		Tavern *game.Tavern `json:"tavern"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	a.ID = aux.ID
	a.Name = aux.Name
	a.Tavern = aux.Tavern

	a.Gold = &atomic.Int64{}
	a.Gold.Store(aux.Gold)

	return nil
}
