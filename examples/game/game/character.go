package game

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"sync"

	"github.com/alfreddobradi/actors/pkg/telemetry"
	"github.com/google/uuid"
)

const (
	xpExponent = 1.7
)

func xpForLevel(level int) int {
	return int(math.Floor(math.Pow(float64(level-1), xpExponent) * 100))
}

const (
	StatusIdle uint8 = iota
	StatusBusy
)

type Inventory struct {
	mx        *sync.Mutex
	resources map[string]int
}

func NewInventory() *Inventory {
	return &Inventory{
		mx:        &sync.Mutex{},
		resources: make(map[string]int),
	}
}

func (inv *Inventory) UnmarshalJSON(data []byte) error {
	var aux struct {
		Resources map[string]int `json:"resources"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	if inv.mx == nil {
		inv.mx = &sync.Mutex{}
	}

	inv.mx.Lock()
	defer inv.mx.Unlock()
	inv.resources = aux.Resources
	return nil
}

func (inv *Inventory) MarshalJSON() ([]byte, error) {
	inv.mx.Lock()
	defer inv.mx.Unlock()

	aux := struct {
		Resources map[string]int `json:"resources"`
	}{
		Resources: inv.resources,
	}
	return json.Marshal(aux)
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

func (inv *Inventory) GetResource(resource Resource) int {
	inv.mx.Lock()
	defer inv.mx.Unlock()
	if inv.resources == nil {
		return 0
	}
	return inv.resources[resource.Name]
}

type Character struct {
	ID         uuid.UUID `json:"id"`
	Name       string    `json:"name"`
	Level      int       `json:"level"`
	Experience int       `json:"experience"`
	Status     uint8     `json:"status"`
	Cooldown   int       `json:"cooldown"`
	Action     Action    `json:"-"`

	Inventory *Inventory `json:"inventory"`
}

func (c Character) MarshalJSON() ([]byte, error) {
	raw := make(map[string]interface{})

	raw["id"] = c.ID
	raw["name"] = c.Name
	raw["level"] = c.Level
	raw["experience"] = c.Experience
	raw["status"] = c.Status
	raw["cooldown"] = c.Cooldown
	raw["inventory"] = c.Inventory
	if c.Action != nil {
		rawAction, err := json.Marshal(c.Action)
		if err != nil {
			return nil, err
		}
		var actionData map[string]interface{}
		if err := json.Unmarshal(rawAction, &actionData); err != nil {
			return nil, err
		}
		actionData["_name"] = c.Action.GetName()
		raw["action"] = actionData
	} else {
		raw["action"] = nil
	}

	return json.Marshal(raw)
}

func (c *Character) UnmarshalJSON(data []byte) error {
	var aux struct {
		ID         uuid.UUID
		Name       string
		Level      int
		Experience int
		Status     uint8
		Cooldown   int
		Action     json.RawMessage
		Inventory  Inventory
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	c.ID = aux.ID
	c.Name = aux.Name
	c.Level = aux.Level
	c.Experience = aux.Experience
	c.Status = aux.Status
	c.Cooldown = aux.Cooldown
	c.Inventory = &aux.Inventory

	if len(aux.Action) > 0 {
		var actionMap map[string]interface{}
		if err := json.Unmarshal(aux.Action, &actionMap); err != nil {
			return err
		}
		switch actionMap["_name"] {
		case "fight":
			var fightAction FightAction
			if err := json.Unmarshal(aux.Action, &fightAction); err != nil {
				return err
			}
			c.Action = &fightAction
		case "gather":
			var gatherAction GatherAction
			if err := json.Unmarshal(aux.Action, &gatherAction); err != nil {
				return err
			}
			c.Action = &gatherAction
		default:
			return fmt.Errorf("unknown action type: %s", actionMap["_name"])
		}
	} else {
		c.Action = nil
	}

	return nil
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

func (c *Character) GainExperience(amount int) {
	slog.Debug("Character is gaining experience", "characterID", c.ID, "characterName", c.Name, "amount", amount, "currentExperience", c.Experience)
	c.Experience += amount
	for i := 0; c.Experience >= xpForLevel(i+1); i++ {
		c.Level = i + 1
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

func (c *Character) gather(ctx context.Context) {
	spanID := telemetry.SpanIDFromContext(ctx)
	ctxLogger := slog.With("span_id", spanID, "characterID", c.ID, "characterName", c.Name)

	action := &GatherAction{Resource: Wood}
	ctxLogger.Info("Character is performing mine action", "resource", action.Resource.Name)
	action.Execute(ctx, c)
}
