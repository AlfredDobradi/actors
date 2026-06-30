package handler

import (
	"net/http"

	"github.com/alfreddobradi/actors/cmd/game/api/state"
)

func NotImplementedHandler(s *state.Context) func(w http.ResponseWriter, r *http.Request) { //nolint:unused
	return func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Not implemented", http.StatusNotImplemented)
	}
}
