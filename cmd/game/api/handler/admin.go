package handler

import (
	"encoding/json"
	"net/http"

	"github.com/alfreddobradi/actors/cmd/game/api/state"
	"github.com/alfreddobradi/actors/cmd/game/repository"
	"github.com/alfreddobradi/actors/pkg/telemetry"
	"github.com/gorilla/mux"
)

func HandleAdminGetAccounts(s *state.Context) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		span := telemetry.SpanFromRequest(r)
		accounts, err := repository.GetAccounts(span.Context(), s.DB)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		w.Header().Set("content-type", "application/json")
		encoder := json.NewEncoder(w)
		if err := encoder.Encode(accounts); err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
	}
}

func HandleAdminGetAccount(s *state.Context) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		span := telemetry.SpanFromRequest(r)

		vars := mux.Vars(r)
		accountID := vars["accountId"]

		account, err := repository.GetAccount(span.Context(), s.DB, accountID)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		w.Header().Set("content-type", "application/json")
		encoder := json.NewEncoder(w)
		if err := encoder.Encode(account); err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
	}
}
