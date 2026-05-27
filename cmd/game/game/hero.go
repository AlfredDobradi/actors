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

type Tavern struct {
	mx *sync.Mutex

	name       string
	characters map[uuid.UUID]*Hero
}

func NewTavern(name string) *Tavern {
	return &Tavern{
		mx: &sync.Mutex{},

		name:       name,
		characters: make(map[uuid.UUID]*Hero),
	}
}

func (t *Tavern) Name() string {
	return t.name
}

func (t *Tavern) UnmarshalJSON(data []byte) error {
	var aux struct {
		Name       string             `json:"name"`
		Characters map[uuid.UUID]Hero `json:"characters"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	if t.mx == nil {
		t.mx = &sync.Mutex{}
	}

	t.mx.Lock()
	defer t.mx.Unlock()

	t.characters = make(map[uuid.UUID]*Hero)
	for id, character := range aux.Characters {
		t.characters[id] = &character
	}
	t.name = aux.Name
	return nil
}

func (t *Tavern) MarshalJSON() ([]byte, error) {
	t.mx.Lock()
	defer t.mx.Unlock()

	characters := make(map[uuid.UUID]Hero)
	for id, character := range t.characters {
		characters[id] = *character
	}

	aux := struct {
		Name       string             `json:"name"`
		Characters map[uuid.UUID]Hero `json:"characters"`
	}{
		Name:       t.name,
		Characters: characters,
	}
	return json.Marshal(aux)
}

func (t *Tavern) AddCharacter(character *Hero) {
	t.mx.Lock()
	defer t.mx.Unlock()
	t.characters[character.ID] = character
}

func (t *Tavern) Characters() map[uuid.UUID]*Hero {
	t.mx.Lock()
	defer t.mx.Unlock()
	return t.characters
}

func (t *Tavern) GetCharacter(characterID uuid.UUID) (*Hero, bool) {
	t.mx.Lock()
	defer t.mx.Unlock()
	if character, exists := t.characters[characterID]; exists {
		return character, true
	}
	return nil, false
}

func (t *Tavern) ProcessTick(ctx context.Context) {
	t.mx.Lock()
	defer t.mx.Unlock()

	if len(t.characters) == 0 {
		return
	}

	wg := sync.WaitGroup{}
	for id, character := range t.characters {
		wg.Add(1)
		go func(id uuid.UUID, character *Hero) {
			defer wg.Done()
			slog.Debug("Processing tick for character", "characterID", character.ID, "characterName", character.Name)
			character.ProcessTick(ctx)
			t.characters[id] = character
		}(id, character)
	}
	wg.Wait()
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

type Hero struct {
	ID         uuid.UUID `json:"id"`
	Name       string    `json:"name"`
	Level      int       `json:"level"`
	Experience int       `json:"experience"`
	Status     uint8     `json:"status"`
	Cooldown   int       `json:"cooldown"`
	Health     int       `json:"health"`
	Energy     int       `json:"energy"`
	Gold       int       `json:"gold"`
	Action     Action    `json:"-"`

	Inventory *Inventory `json:"inventory"`
}

func (c Hero) MarshalJSON() ([]byte, error) {
	raw := make(map[string]interface{})

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

func (c *Hero) fight(ctx context.Context) { //nolint:unused
	spanID := telemetry.SpanIDFromContext(ctx)
	ctxLogger := slog.With("span_id", spanID, "characterID", c.ID, "characterName", c.Name)

	action := &FightAction{}
	ctxLogger.Info("Character is performing fight action")
	action.Execute(ctx, c)
}

func (c *Hero) gather(ctx context.Context) { //nolint:unused
	spanID := telemetry.SpanIDFromContext(ctx)
	ctxLogger := slog.With("span_id", spanID, "characterID", c.ID, "characterName", c.Name)

	action := &GatherAction{Resource: Wood}
	ctxLogger.Info("Character is performing mine action", "resource", action.Resource.Name)
	action.Execute(ctx, c)
}
