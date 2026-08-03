package game

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"strings"

	"github.com/alfreddobradi/actors/pkg/telemetry"
)

const (
	TickRate = 1 // Tick every second
)

const (
	FightBaseCooldown  = 10
	GatherBaseCooldown = 5
)

type Action interface {
	GetName() string
	Execute(ctx context.Context, character *Hero, guild *Guild)
	GetCooldown() int
	String() string
}

var ActionMap = map[string]Action{
	ActionNameAdventure: &AdventureAction{},
	ActionNameHeal:      &HealAction{},
	ActionNameRest:      &RestAction{},
	ActionNameIdle:      &IdleAction{},
	ActionNameTavern:    &TavernAction{},
	ActionNameGather:    &GatherAction{},
}

const (
	ActionNameAdventure = "adventure"
	ActionNameHeal      = "heal"
	ActionNameRest      = "rest"
	ActionNameIdle      = "idle"
	ActionNameTavern    = "tavern"
	ActionNameGather    = "gather"
)

type chanceEntry struct {
	action Action
	weight int
}

func debugChanceTable(table []chanceEntry) string {
	steps := make([]int, len(table))
	for i := range table {
		if i == 0 {
			steps[i] = table[i].weight
		} else {
			steps[i] = steps[i-1] + table[i].weight
		}
	}

	b := strings.Builder{}
	for i, entry := range table {
		b.WriteString(strings.TrimSpace(
			fmt.Sprintf("%s: %d", entry.action.GetName(), steps[i]),
		))
		b.WriteString(", ")
	}

	return b.String()
}

func (c *Hero) WhatNext() Action {
	if c.Health < 50 {
		slog.Debug("Hero is low on health, deciding to heal", "characterID", c.ID, "characterName", c.Name, "health", c.Health)
		return &HealAction{}
	}

	if c.Energy < 50 {
		slog.Debug("Hero is low on energy, deciding to rest", "characterID", c.ID, "characterName", c.Name, "energy", c.Energy)
		return &RestAction{}
	}

	chanceTable := []chanceEntry{
		{action: &AdventureAction{}, weight: 40},
		{action: &GatherAction{}, weight: 30},
		{action: &IdleAction{}, weight: 10},
	}

	if c.Gold > 20 {
		chanceTable = append(chanceTable, chanceEntry{
			action: &TavernAction{},
			weight: 20,
		})
	}

	totalWeight := 0
	for _, entry := range chanceTable {
		totalWeight += entry.weight
	}

	roll := rand.Intn(totalWeight) //nolint:gosec

	slog.Info("Deciding what to do...", "table", debugChanceTable(chanceTable), "roll", roll)

	var action Action
	for _, entry := range chanceTable {
		if roll < entry.weight {
			action = entry.action
			break
		}
		roll -= entry.weight
	}

	if action.GetName() == ActionNameGather && action.(*GatherAction).Resource.Name == "" {
		roll := rand.Intn(100) //nolint:gosec
		if roll < 50 {
			action = &GatherAction{Resource: Wood}
		} else if roll < 80 {
			action = &GatherAction{Resource: Stone}
		} else {
			action = &GatherAction{Resource: Iron}
		}
	}

	if action == nil {
		return &IdleAction{}
	}
	return action
}

type FightAction struct{}

func (f *FightAction) GetName() string {
	return "fight"
}

func (f *FightAction) String() string {
	return "is fighting"
}

func (f *FightAction) Execute(ctx context.Context, character *Hero, _ *Guild) {
	spanID := telemetry.SpanIDFromContext(ctx)
	ctxLogger := slog.With("span_id", spanID, "characterID", character.ID, "characterName", character.Name)
	ctxLogger.Info("Executing fight action")

	lowerBound := max(character.Level-5, 0)
	upperBound := character.Level + 5
	enemyRoll := rand.Intn(upperBound-lowerBound) + lowerBound //nolint:gosec
	success := enemyRoll <= character.Level
	ctxLogger.Info("Resolving fight", "characterLevel", character.Level, "enemyLevelRoll", enemyRoll, "success", success)
	if success {
		character.GainExperience(10)
		ctxLogger.Debug("Added experience to character", "newExperience", character.Experience)
	}
}

func (f *FightAction) GetCooldown() int {
	return FightBaseCooldown
}

func (g *GatherAction) GetName() string {
	return "gather"
}

func (g *GatherAction) String() string {
	return "is gathering " + g.Resource.Name
}

type GatherAction struct {
	Resource Resource
}

func (g *GatherAction) Execute(ctx context.Context, character *Hero, _ *Guild) {
	spanID := telemetry.SpanIDFromContext(ctx)
	ctxLogger := slog.With("span_id", spanID, "characterID", character.ID, "characterName", character.Name)
	ctxLogger.Info("Executing gather action", "resource", g.Resource.Name)

	gatherChance := rand.Int31n(100) //nolint:gosec
	if g.Resource.Gather(ctx, gatherChance) {
		batchSize := g.Resource.Batch(ctx)
		character.Inventory.AddResource(g.Resource, batchSize)
		experienceGained := int(float64(batchSize) * g.Resource.Experience)
		character.GainExperience(experienceGained)
		ctxLogger.Info("Successfully gathered resource", "resource", g.Resource.Name, "quantity", batchSize, "experienceGained", experienceGained, "newExperience", character.Experience)
	}
}

func (g *GatherAction) GetCooldown() int {
	return int(float64(GatherBaseCooldown) * g.Resource.CooldownMultiplier)
}

type HealAction struct{}

func (h *HealAction) GetName() string {
	return ActionNameHeal
}

func (h *HealAction) String() string {
	return "is healing"
}

func (h *HealAction) Execute(ctx context.Context, character *Hero, _ *Guild) {
	spanID := telemetry.SpanIDFromContext(ctx)
	ctxLogger := slog.With("span_id", spanID, "characterID", character.ID, "characterName", character.Name)
	ctxLogger.Info("Executing heal action")

	character.Health = 100
	ctxLogger.Info("Character healed")
}

func (h *HealAction) GetCooldown() int {
	return 10
}

type RestAction struct{}

func (r *RestAction) GetName() string {
	return ActionNameRest
}

func (r *RestAction) String() string {
	return "is resting"
}

func (r *RestAction) Execute(ctx context.Context, character *Hero, _ *Guild) {
	spanID := telemetry.SpanIDFromContext(ctx)
	ctxLogger := slog.With("span_id", spanID, "characterID", character.ID, "characterName", character.Name)
	ctxLogger.Info("Executing rest action")

	character.Energy = 100
	ctxLogger.Info("Character rested")
}

func (r *RestAction) GetCooldown() int {
	return 10
}

type IdleAction struct{}

func (i *IdleAction) GetName() string {
	return ActionNameIdle
}

func (i *IdleAction) String() string {
	return "is doing absolutely nothing"
}

func (i *IdleAction) Execute(ctx context.Context, character *Hero, _ *Guild) {
	spanID := telemetry.SpanIDFromContext(ctx)
	ctxLogger := slog.With("span_id", spanID, "characterID", character.ID, "characterName", character.Name)
	ctxLogger.Info("Character is idle")
}

func (i *IdleAction) GetCooldown() int {
	return 1
}

type TavernAction struct{}

func (t *TavernAction) GetName() string {
	return ActionNameTavern
}

func (t *TavernAction) String() string {
	return "is visiting the tavern"
}

func (t *TavernAction) Execute(ctx context.Context, hero *Hero, g *Guild) {
	spanID := telemetry.SpanIDFromContext(ctx)
	ctxLogger := slog.With("span_id", spanID, "characterID", hero.ID, "characterName", hero.Name)
	ctxLogger.Info("Character is visiting the tavern")

	moneyModifier := rand.Float64()*100 - 100 //nolint:gosec
	receipt := hero.GainGold(moneyModifier, g.settings.TaxRate)
	g.Gold.Add(receipt.Tax)

	energyLoss := rand.Intn(25) + 5 //nolint:gosec
	hero.Energy -= energyLoss
	ctxLogger.Info("Character's tavern visit resulted in gold change", "gold_change", receipt.Net, "new_gold", hero.Gold, "energy_loss", energyLoss, "new_energy", hero.Energy)
}

func (t *TavernAction) GetCooldown() int {
	return 10
}

type AdventureAction struct{}

func (a *AdventureAction) GetName() string {
	return ActionNameAdventure
}

func (a *AdventureAction) String() string {
	return "is going on an adventure"
}

func (a *AdventureAction) Execute(ctx context.Context, hero *Hero, g *Guild) {
	spanID := telemetry.SpanIDFromContext(ctx)
	ctxLogger := slog.With("span_id", spanID, "characterID", hero.ID, "characterName", hero.Name)
	ctxLogger.Info("Character is going on an adventure")

	// For simplicity, we'll just have a random chance to gain experience and lose some health and energy
	experienceGained := rand.Intn(50) + 10 //nolint:gosec
	healthLoss := rand.Intn(30) + 10       //nolint:gosec
	energyLoss := rand.Intn(30) + 10       //nolint:gosec
	goldGained := rand.Float64() * 100     //nolint:gosec

	hero.GainExperience(experienceGained)
	hero.Health -= healthLoss
	hero.Energy -= energyLoss
	receipt := hero.GainGold(goldGained, g.settings.TaxRate)
	g.Gold.Add(receipt.Tax)

	ctxLogger.Info("Character's adventure results", "experience_gained", experienceGained, "new_experience", hero.Experience, "health_loss", healthLoss, "new_health", hero.Health, "energy_loss", energyLoss, "new_energy", hero.Energy, "gold_gained", receipt.Net, "new_gold", hero.Gold)
}

func (a *AdventureAction) GetCooldown() int {
	return 15
}
