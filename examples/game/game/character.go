package game

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/alfreddobradi/actors/examples/game/telemetry"
	"github.com/google/uuid"
)

const (
	StatusIdle uint8 = iota
	StatusBusy
)

type Inventory struct {
	mx        *sync.Mutex
	resources map[string]int
}

func NewInventory() Inventory {
	return Inventory{
		mx:        &sync.Mutex{},
		resources: make(map[string]int),
	}
}

func (inv *Inventory) AddResource(resource Resource, quantity int) {
	inv.mx.Lock()
	defer inv.mx.Unlock()
	if inv.resources == nil {
		inv.resources = make(map[string]int)
	}

	if _, exists := inv.resources[resource.Name]; !exists {
		inv.resources[resource.Name] = 0
	}

	inv.resources[resource.Name] += quantity
}

type Character struct {
	ID         uuid.UUID
	Name       string
	Level      int
	Experience int
	Status     uint8
	Cooldown   int
	Action     Action

	Inventory
}

func NewCharacter(name string) Character {
	return Character{
		ID:         uuid.New(),
		Name:       name,
		Level:      1,
		Experience: 0,
		Status:     StatusIdle,
		Cooldown:   0,
		Inventory:  NewInventory(),
	}
}

func (c *Character) ProcessTick(ctx context.Context) {
	spanID := telemetry.SpanIDFromContext(ctx)
	ctxLogger := slog.With("span_id", spanID, "characterID", c.ID, "characterName", c.Name)

	if c.Action == nil {
		return
	}

	if c.Cooldown > 0 {
		c.Cooldown--
		ctxLogger.Info("Character is busy, skipping tick processing", "cooldown", c.Cooldown)
		return
	}

	switch a := c.Action.(type) {
	case *FightAction:
		a.Execute(ctx, c)
		c.Cooldown = a.GetCooldown()
	case *GatherAction:
		a.Execute(ctx, c)
		c.Cooldown = a.GetCooldown()
	default:
		ctxLogger.Debug("No action assigned to character, idle")
	}
}

func (c *Character) StartAction(ctx context.Context, action Action) {
	spanID := telemetry.SpanIDFromContext(ctx)

	if c.Action != nil && c.Action.GetName() == action.GetName() {
		slog.Debug("Character is already performing this action", "span_id", spanID, "characterID", c.ID, "characterName", c.Name, "actionType", fmt.Sprintf("%T", action))
		return
	}

	slog.Debug("Character is starting an action", "span_id", spanID, "characterID", c.ID, "characterName", c.Name, "actionType", fmt.Sprintf("%T", action))

	c.Action = action
	c.Cooldown = action.GetCooldown()
}

func (c *Character) StopAction(ctx context.Context) {
	if c.Action == nil {
		return
	}

	spanID := telemetry.SpanIDFromContext(ctx)
	slog.Info("Character is stopping current action", "span_id", spanID, "characterID", c.ID, "characterName", c.Name)

	c.Action = nil
	c.Cooldown = 0
}

func (c *Character) fight(ctx context.Context) {
	spanID := telemetry.SpanIDFromContext(ctx)
	ctxLogger := slog.With("span_id", spanID, "characterID", c.ID, "characterName", c.Name)

	action := &FightAction{}
	ctxLogger.Info("Character is performing fight action")
	action.Execute(ctx, c)
}

func (c *Character) mine(ctx context.Context) {
	spanID := telemetry.SpanIDFromContext(ctx)
	ctxLogger := slog.With("span_id", spanID, "characterID", c.ID, "characterName", c.Name)

	action := &GatherAction{Resource: Wood}
	ctxLogger.Info("Character is performing mine action", "resource", action.Resource.Name)
	action.Execute(ctx, c)
}
