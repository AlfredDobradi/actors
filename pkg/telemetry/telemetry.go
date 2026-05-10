package telemetry

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/alfreddobradi/actors/pkg/model"
	"github.com/google/uuid"
)

type Span struct {
	id      uuid.UUID
	context context.Context
}

func (s *Span) GetLogger() *slog.Logger {
	return slog.With("span_id", s.id.String())
}

func (s *Span) Context() context.Context {
	return s.context
}

func SpanIDFromContext(ctx context.Context) uuid.UUID {
	if spanID, ok := ctx.Value(model.ContextKeySpanID).(uuid.UUID); ok {
		return spanID
	}

	return uuid.New()
}

func SpanIDFromRequest(r *http.Request) uuid.UUID {
	if spanID, ok := r.Context().Value(model.ContextKeySpanID).(uuid.UUID); ok {
		return spanID
	}

	return uuid.New()
}

func SpanFromRequest(r *http.Request) *Span {
	spanID := SpanIDFromRequest(r)
	ctx := context.WithValue(r.Context(), model.ContextKeySpanID, spanID)
	return &Span{id: spanID, context: ctx}
}

func SpanFromContext(ctx context.Context) *Span {
	spanID := SpanIDFromContext(ctx)
	ctx = context.WithValue(ctx, model.ContextKeySpanID, spanID)
	return &Span{id: spanID, context: ctx}
}
