package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/alfreddobradi/actors/cmd/game/api/state"
	"github.com/alfreddobradi/actors/cmd/game/model"
	"github.com/alfreddobradi/actors/pkg/system"
	"github.com/alfreddobradi/actors/pkg/telemetry"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

func HandleCreateTavern(s *state.Context) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		span := telemetry.SpanFromRequest(r)

		accountData, ok := r.Context().Value(model.ContextKeyAccountData).(*model.Account)
		if !ok || accountData == nil {
			http.Error(w, "Failed to retrieve account data from context", http.StatusInternalServerError)
			return
		}

		httpReq := model.NewTavernRequest{}
		if err := json.NewDecoder(r.Body).Decode(&httpReq); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		defer closeBody(r.Body)

		_, err := s.System.Request(span.Context(), uuid.Nil, system.Recipient{Kind: system.RecipientKindActor, Subject: accountData.ID.String()}, httpReq)
		if err != nil {
			http.Error(w, "Failed to request tavern creation", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("Tavern creation requested successfully")); err != nil {
			slog.Warn("Failed to write response", "error", err)
		}
	}
}

func HandleHireCharacter(s *state.Context) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		span := telemetry.SpanFromRequest(r)

		accountData, ok := r.Context().Value(model.ContextKeyAccountData).(*model.Account)
		if !ok || accountData == nil {
			http.Error(w, "Failed to retrieve account data from context", http.StatusInternalServerError)
			return
		}

		message := model.HireCharacterRequest{}
		resp, err := s.System.Request(span.Context(), uuid.Nil, system.Recipient{Kind: system.RecipientKindActor, Subject: accountData.ID.String()}, message)
		if err != nil {
			http.Error(w, "Failed to request character hire", http.StatusInternalServerError)
			return
		}

		response, ok := resp.(model.HireCharacterResponse)
		if !ok {
			http.Error(w, "Invalid response from character hire request", http.StatusInternalServerError)
			return
		}

		if !response.OK {
			http.Error(w, "Failed to hire character: "+response.Error, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, "Failed to encode hire character response", http.StatusInternalServerError)
		}
	}
}

func HandleGetCharacter(s *state.Context) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		span := telemetry.SpanFromRequest(r)

		vars := mux.Vars(r)
		characterID, ok := vars["characterId"]
		if !ok {
			http.Error(w, "Character ID is required", http.StatusBadRequest)
			return
		}

		accountData, ok := r.Context().Value(model.ContextKeyAccountData).(*model.Account)
		if !ok || accountData == nil {
			http.Error(w, "Failed to retrieve account data from context", http.StatusInternalServerError)
			return
		}

		req := model.GetCharacterRequest{
			ID: characterID,
		}

		characterData, err := s.System.Request(span.Context(), uuid.Nil, system.Recipient{Kind: system.RecipientKindActor, Subject: accountData.ID.String()}, req)
		if err != nil {
			http.Error(w, "Failed to request character", http.StatusInternalServerError)
			return
		}

		data, ok := characterData.(model.GetCharacterResponse)
		if !ok {
			http.Error(w, "Invalid response from character request", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(data); err != nil {
			http.Error(w, "Failed to encode character details", http.StatusInternalServerError)
			return
		}
	}
}

func HandleGetCharacters(s *state.Context) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		span := telemetry.SpanFromRequest(r)

		accountData, ok := r.Context().Value(model.ContextKeyAccountData).(*model.Account)
		if !ok || accountData == nil {
			http.Error(w, "Failed to retrieve account data from context", http.StatusInternalServerError)
			return
		}

		req := model.GetCharactersRequest{}

		characterData, err := s.System.Request(span.Context(), uuid.Nil, system.Recipient{Kind: system.RecipientKindActor, Subject: accountData.ID.String()}, req)
		if err != nil {
			http.Error(w, "Failed to request character", http.StatusInternalServerError)
			return
		}

		data, ok := characterData.(model.GetCharactersResponse)
		if !ok {
			http.Error(w, "Invalid response from character request", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(data); err != nil {
			http.Error(w, "Failed to encode character details", http.StatusInternalServerError)
			return
		}
	}
}
