package game

import (
	"context"
	"log/slog"
	"math/rand"

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
	Execute(ctx context.Context, character *Character)
	GetCooldown() int
	String() string
}

type FightAction struct{}

func (f *FightAction) GetName() string {
	return "fight"
}

func (f *FightAction) String() string {
	return "is fighting"
}

func (f *FightAction) Execute(ctx context.Context, character *Character) {
	spanID := telemetry.SpanIDFromContext(ctx)
	ctxLogger := slog.With("span_id", spanID, "characterID", character.ID, "characterName", character.Name)
	ctxLogger.Info("Executing fight action")

	lowerBound := max(character.Level-5, 0)
	upperBound := character.Level + 5
	enemyRoll := rand.Intn(upperBound-lowerBound) + lowerBound
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

func (g *GatherAction) Execute(ctx context.Context, character *Character) {
	spanID := telemetry.SpanIDFromContext(ctx)
	ctxLogger := slog.With("span_id", spanID, "characterID", character.ID, "characterName", character.Name)
	ctxLogger.Info("Executing gather action", "resource", g.Resource.Name)

	gatherChance := rand.Int31n(100)
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
