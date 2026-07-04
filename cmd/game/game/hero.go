package game

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"math/rand"
	"sync"
	"sync/atomic"

	"github.com/alfreddobradi/actors/pkg/telemetry"
	"github.com/google/uuid"
)

func HeroPriceMultiplier(heroAmount int) int64 {
	bands := [][2]int{
		{3, 1},
		{6, 2},
		{12, 3},
		{21, 4},
	}

	for _, band := range bands {
		amount, multiplier := band[0], band[1]
		if heroAmount <= int(amount) {
			return int64(multiplier)
		}
	}
	return 5
}

type Guild struct {
	mx *sync.RWMutex

	id     uuid.UUID
	name   string
	heroes map[uuid.UUID]*Hero
	Gold   *atomic.Int64
}

func NewGuild(name string) *Guild {
	startingMoney := &atomic.Int64{}
	startingMoney.Store(3000)

	return &Guild{
		mx: &sync.RWMutex{},

		id:     uuid.New(),
		name:   name,
		heroes: make(map[uuid.UUID]*Hero),
		Gold:   startingMoney,
	}
}

func (g *Guild) Name() string {
	return g.name
}

func (g *Guild) ID() uuid.UUID {
	return g.id
}

func (g *Guild) Heroes() map[uuid.UUID]*Hero {
	g.mx.RLock()
	defer g.mx.RUnlock()
	return g.heroes
}

func (g *Guild) UnmarshalJSON(data []byte) error {
	var aux struct {
		Name       string             `json:"name"`
		Characters map[uuid.UUID]Hero `json:"characters"`
		Gold       int64              `json:"gold"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	if g.mx == nil {
		g.mx = &sync.RWMutex{}
	}

	g.mx.Lock()
	defer g.mx.Unlock()

	g.heroes = make(map[uuid.UUID]*Hero)
	for id, character := range aux.Characters {
		g.heroes[id] = &character
	}
	g.name = aux.Name

	if g.Gold == nil {
		g.Gold = &atomic.Int64{}
	}
	g.Gold.Store(aux.Gold)

	return nil
}

func (g *Guild) MarshalJSON() ([]byte, error) {
	g.mx.RLock()
	defer g.mx.RUnlock()

	characters := make(map[uuid.UUID]Hero)
	for id, character := range g.heroes {
		characters[id] = *character
	}

	gold := int64(0)
	if g.Gold != nil {
		gold = g.Gold.Load()
	}

	aux := struct {
		Name       string             `json:"name"`
		Characters map[uuid.UUID]Hero `json:"characters"`
		Gold       int64              `json:"gold"`
	}{
		Name:       g.name,
		Characters: characters,
		Gold:       gold,
	}
	return json.Marshal(aux)
}

func (g *Guild) AddCharacter(character *Hero) {
	g.mx.Lock()
	defer g.mx.Unlock()
	g.heroes[character.ID] = character
}

func (g *Guild) Characters() map[uuid.UUID]*Hero {
	g.mx.RLock()
	defer g.mx.RUnlock()
	return g.heroes
}

func (g *Guild) GetCharacter(characterID uuid.UUID) (*Hero, bool) {
	g.mx.RLock()
	defer g.mx.RUnlock()
	if character, exists := g.heroes[characterID]; exists {
		return character, true
	}
	return nil, false
}

func (g *Guild) ProcessTick(ctx context.Context) {
	g.mx.RLock()
	if len(g.heroes) == 0 {
		g.mx.RUnlock()
		return
	}
	g.mx.RUnlock()

	wg := sync.WaitGroup{}
	for id, character := range g.heroes {
		wg.Add(1)
		go func(id uuid.UUID, character *Hero) {
			defer wg.Done()
			slog.Debug("Processing tick for character", "characterID", character.ID, "characterName", character.Name)
			character.ProcessTick(ctx)

			g.mx.Lock()
			g.heroes[id] = character
			g.mx.Unlock()
		}(id, character)
	}
	wg.Wait()
}

func (g *Guild) ReplayTicks(ctx context.Context, ticks int) []error {
	wg := sync.WaitGroup{}
	errors := make(chan error, len(g.heroes))
	for id, character := range g.heroes {
		wg.Add(1)
		go func(id uuid.UUID, character *Hero) {
			defer wg.Done()
			slog.Debug("Replaying ticks for character", "characterID", character.ID, "characterName", character.Name, "ticks", ticks)
			// There is currently no way of receiving an error from the hero.ProcessTick method, might change in the future.
			if err := character.ReplayTicks(ctx, ticks); err != nil {
				errors <- fmt.Errorf("error replaying ticks for character %s: %w", character.ID, err)
				return
			}

			g.mx.Lock()
			g.heroes[id] = character
			g.mx.Unlock()
		}(id, character)
	}
	wg.Wait()

	close(errors)
	if len(errors) > 0 {
		errs := make([]error, 0, len(errors))
		for err := range errors {
			errs = append(errs, err)
		}
		return errs
	}
	return nil
}

func GenerateRandomCharacter() Hero {
	firstNames := []string{"Arin", "Bel", "Cal", "Dain", "Eli"}
	lastNames := []string{"Strong", "Swift", "Brave", "Clever", "Bold"}

	name := fmt.Sprintf("%s %s", firstNames[rand.Intn(len(firstNames))], lastNames[rand.Intn(len(lastNames))]) //nolint:gosec
	character := NewHero(name)
	return character
}

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

func (inv *Inventory) Resources() map[string]int {
	inv.mx.Lock()
	defer inv.mx.Unlock()
	return inv.resources
}

type Hero struct {
	ID         uuid.UUID `json:"id" db:"id"`
	Name       string    `json:"name" db:"name"`
	Level      int       `json:"level" db:"level"`
	Experience int       `json:"experience" db:"experience"`
	Status     uint8     `json:"status" db:"status"`
	Cooldown   int       `json:"cooldown" db:"cooldown"`
	Health     int       `json:"health" db:"health"`
	Energy     int       `json:"energy" db:"energy"`
	Gold       int       `json:"gold" db:"gold"`
	Action     Action    `json:"-" db:"-"`

	Inventory *Inventory `json:"inventory" db:"inventory"`
}

func (c Hero) MarshalJSON() ([]byte, error) {
	raw := make(map[string]any)

	raw["id"] = c.ID
	raw["name"] = c.Name
	raw["level"] = c.Level
	raw["experience"] = c.Experience
	raw["status"] = c.Status
	raw["cooldown"] = c.Cooldown
	raw["health"] = c.Health
	raw["energy"] = c.Energy
	raw["gold"] = c.Gold
	raw["inventory"] = c.Inventory
	if c.Action != nil {
		rawAction, err := json.Marshal(c.Action)
		if err != nil {
			return nil, err
		}
		var actionData map[string]any
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

func (c *Hero) UnmarshalJSON(data []byte) error {
	var aux struct {
		ID         uuid.UUID
		Name       string
		Level      int
		Experience int
		Status     uint8
		Cooldown   int
		Health     int
		Energy     int
		Gold       int
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
	c.Health = aux.Health
	c.Energy = aux.Energy
	c.Gold = aux.Gold
	c.Inventory = &aux.Inventory

	if len(aux.Action) > 0 {
		if bytes.Equal(aux.Action, []byte("null")) {
			c.Action = nil
			return nil
		}

		var actionRaw map[string]interface{}
		if err := json.Unmarshal(aux.Action, &actionRaw); err != nil {
			return err
		}
		actionName, ok := actionRaw["_name"].(string)
		if !ok {
			return fmt.Errorf("action data missing _name field")
		}

		decodedAction, exists := actionMap[actionName]
		if !exists {
			return fmt.Errorf("unknown action type: %s", actionName)
		}

		if err := json.Unmarshal(aux.Action, &decodedAction); err != nil {
			return err
		}

		c.Action = decodedAction
	} else {
		c.Action = nil
	}

	return nil
}

func NewHero(name string) Hero {
	return Hero{
		ID:         uuid.New(),
		Name:       name,
		Level:      1,
		Experience: 0,
		Status:     StatusIdle,
		Cooldown:   0,
		Health:     100,
		Energy:     100,
		Gold:       100,
		Inventory:  NewInventory(),
		Action:     &IdleAction{},
	}
}

func (c *Hero) GainExperience(amount int) {
	slog.Debug("Character is gaining experience", "characterID", c.ID, "characterName", c.Name, "amount", amount, "currentExperience", c.Experience)
	c.Experience += amount
	for i := 0; c.Experience >= xpForLevel(i+1); i++ {
		c.Level = i + 1
	}
}

func (c *Hero) ProcessTick(ctx context.Context) {
	spanID := telemetry.SpanIDFromContext(ctx)
	ctxLogger := slog.With("span_id", spanID, "characterID", c.ID, "characterName", c.Name)

	if c.Action == nil || c.Action.GetName() == ActionNameIdle {
		c.Action = c.WhatNext()

		c.Cooldown = c.Action.GetCooldown()
		return
	}

	if c.Cooldown > 0 {
		c.Cooldown--
		ctxLogger.Info("Character is performing an action", "action", c.Action.GetName(), "ticks_left", c.Cooldown)
		return
	}

	c.Action.Execute(ctx, c)
	c.Action = &IdleAction{}
	c.Cooldown = 0
}

func (c *Hero) ReplayTicks(ctx context.Context, ticks int) error {
	remainingTicks := ticks

	slog.Debug("Starting to replay ticks", "characterID", c.ID, "characterName", c.Name, "ticksToReplay", ticks)

	actions := make([]string, 0)
	if c.Action != nil {
		actions = append(actions, c.Action.GetName())
	}

	for remainingTicks > 0 {
		slog.Debug("replaying ticks", "remaining", remainingTicks)
		if c.Cooldown > 0 && c.Action != nil {
			slog.Debug("cooldown is greater than 0 and there is an action", "cooldown", c.Cooldown, "action", c.Action.GetName())
			newRemainingTicks := max(0, remainingTicks-c.Cooldown)
			c.Cooldown -= remainingTicks - newRemainingTicks
			remainingTicks = newRemainingTicks
			slog.Debug("after processing cooldown", "remaining", remainingTicks, "cooldown", c.Cooldown)
		} else {
			if c.Action != nil {
				slog.Debug("cooldown expired and there is an action to execute", "action", c.Action.GetName())
				c.Action.Execute(ctx, c)
				remainingTicks--
				slog.Debug("after executing action", "remaining", remainingTicks)
			}
			c.Action = c.WhatNext()
			c.Cooldown = c.Action.GetCooldown()
			actions = append(actions, c.Action.GetName())
		}
	}

	slog.Debug("Finished replaying ticks", "characterID", c.ID, "characterName", c.Name, "ticksReplayed", ticks, "actions", actions)

	return nil
}

func (c *Hero) StartAction(ctx context.Context, action Action) {
	spanID := telemetry.SpanIDFromContext(ctx)

	if c.Action != nil && c.Action.GetName() == action.GetName() {
		slog.Debug("Character is already performing this action", "span_id", spanID, "characterID", c.ID, "characterName", c.Name, "actionType", fmt.Sprintf("%T", action))
		return
	}

	slog.Debug("Character is starting an action", "span_id", spanID, "characterID", c.ID, "characterName", c.Name, "actionType", fmt.Sprintf("%T", action))

	c.Action = action
	c.Cooldown = action.GetCooldown()
}

func (c *Hero) StopAction(ctx context.Context) {
	if c.Action == nil {
		return
	}

	spanID := telemetry.SpanIDFromContext(ctx)
	slog.Info("Character is stopping current action", "span_id", spanID, "characterID", c.ID, "characterName", c.Name)

	c.Action = nil
	c.Cooldown = 0
}
