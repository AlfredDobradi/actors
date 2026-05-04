package game

import (
	"context"
	"log/slog"
	"math/rand"

	"github.com/alfreddobradi/actors/examples/game/telemetry"
)

type Resource struct {
	Name               string
	Experience         float64
	Difficulty         float64
	BatchSize          [2]int
	CooldownMultiplier float64
}

func (r Resource) Gather(ctx context.Context, roll int32) bool {
	spanID := telemetry.SpanIDFromContext(ctx)
	ctxLogger := slog.With("span_id", spanID, "resourceName", r.Name)
	target := 100 - int32(r.Difficulty*100)
	ctxLogger.Info("Gather chance roll", "gatherChance", roll, "target", target)
	return roll < target
}

func (r Resource) Batch(ctx context.Context) int {
	spanID := telemetry.SpanIDFromContext(ctx)
	ctxLogger := slog.With("span_id", spanID, "resourceName", r.Name)

	low := r.BatchSize[0]
	high := r.BatchSize[1]
	batchSize := rand.Intn(high-low+1) + low

	ctxLogger.Info("Determined batch size for resource", "resource", r.Name, "low", low, "high", high, "batchSize", batchSize)
	return batchSize
}

var (
	Wood  = Resource{Name: "Wood", Experience: 5, Difficulty: 0.1, BatchSize: [2]int{1, 5}, CooldownMultiplier: 1.0}
	Stone = Resource{Name: "Stone", Experience: 10, Difficulty: 0.25, BatchSize: [2]int{1, 3}, CooldownMultiplier: 2.0}
	Iron  = Resource{Name: "Iron", Experience: 20, Difficulty: 0.50, BatchSize: [2]int{1, 2}, CooldownMultiplier: 4.0}
)

func ResourceByName(name string) (Resource, bool) {
	switch name {
	case "wood":
		return Wood, true
	case "stone":
		return Stone, true
	case "iron":
		return Iron, true
	default:
		return Resource{}, false
	}
}
