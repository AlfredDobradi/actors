package telemetry

import (
	"context"

	"github.com/alfreddobradi/actors/pkg/system"
	"github.com/google/uuid"
)

func SpanIDFromContext(ctx context.Context) uuid.UUID {
	if spanID, ok := ctx.Value(system.ContextKeySpanID).(uuid.UUID); ok {
		return spanID
	}

	return uuid.New()
}
