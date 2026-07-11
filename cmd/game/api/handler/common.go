package handler

import (
	"io"
	"log/slog"
	"net/http"

	"github.com/alfreddobradi/actors/cmd/game/api/state"
)

func closeBody(body io.Closer) {
	if err := body.Close(); err != nil {
		slog.Error("failed to close response body", "error", err)
	}
}

func NotImplementedHandler(s *state.Context) func(w http.ResponseWriter, r *http.Request) { //nolint:unused
	return func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Not implemented", http.StatusNotImplemented)
	}
}
