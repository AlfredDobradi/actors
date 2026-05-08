package api

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/alfreddobradi/actors/examples/game/actor"
	"github.com/alfreddobradi/actors/examples/game/game"
	"github.com/alfreddobradi/actors/examples/game/model"
	"github.com/alfreddobradi/actors/examples/game/paseto"
	"github.com/alfreddobradi/actors/examples/game/repository"
	"github.com/alfreddobradi/actors/examples/game/telemetry"
	"github.com/alfreddobradi/actors/pkg/system"
	"github.com/davecgh/go-spew/spew"
	"github.com/google/uuid"
)

func (s *Server) handleGetCharacter(w http.ResponseWriter, r *http.Request) {
	span := telemetry.SpanFromRequest(r)
	decoder := json.NewDecoder(r.Body)
	var req model.GetCharacterRequest
	if err := decoder.Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	characterData, err := s.sys.Request(span.Context(), uuid.Nil, system.Recipient{Kind: system.RecipientKindTopic, Subject: "character"}, req)
	if err != nil {
		http.Error(w, "Failed to request character", http.StatusInternalServerError)
		return
	}

	spew.Fdump(w, characterData)
}

func (s *Server) handleStartAction(w http.ResponseWriter, r *http.Request) {
	span := telemetry.SpanFromRequest(r)
	decoder := json.NewDecoder(r.Body)
	var httpReq model.StartActionRequest
	if err := decoder.Decode(&httpReq); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var action game.Action
	switch httpReq.Action {
	case "gather":
		if resourceName, ok := httpReq.Context["resource"].(string); ok {
			resource, found := game.ResourceByName(resourceName)
			if !found {
				http.Error(w, "Resource not found", http.StatusBadRequest)
				return
			}
			action = &game.GatherAction{Resource: resource}
		} else {
			http.Error(w, "Invalid resource", http.StatusBadRequest)
			return
		}
	case "fight":
		action = &game.FightAction{}
	default:
		http.Error(w, "Invalid action type", http.StatusBadRequest)
		return
	}

	characterID, err := uuid.Parse(httpReq.CharacterID)
	if err != nil {
		http.Error(w, "Invalid character ID", http.StatusBadRequest)
		return
	}

	message := model.StartActionMessage{
		CharacterID: characterID,
		Action:      action,
		Context:     httpReq.Context,
	}
	if err := s.sys.Publish(span.Context(), uuid.Nil, system.Recipient{Kind: system.RecipientKindTopic, Subject: "character"}, message); err != nil {
		http.Error(w, "Failed to request character action start", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Action " + httpReq.Action + " started successfully"))
}

func (s *Server) handleStopAction(w http.ResponseWriter, r *http.Request) {
	span := telemetry.SpanFromRequest(r)
	decoder := json.NewDecoder(r.Body)
	var httpReq model.StopActionRequest
	if err := decoder.Decode(&httpReq); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	characterID, err := uuid.Parse(httpReq.CharacterID)
	if err != nil {
		http.Error(w, "Invalid character ID", http.StatusBadRequest)
		return
	}

	message := model.StopActionMessage{
		CharacterID: characterID,
	}

	if err := s.sys.Publish(span.Context(), uuid.Nil, system.Recipient{Kind: system.RecipientKindTopic, Subject: "character"}, message); err != nil {
		http.Error(w, "Failed to request character action stop", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Action stopped successfully"))
}

func (s *Server) handleCreateAccount(w http.ResponseWriter, r *http.Request) {
	span := telemetry.SpanFromRequest(r)
	decoder := json.NewDecoder(r.Body)
	var httpReq model.CreateAccountRequest
	if err := decoder.Decode(&httpReq); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if err := repository.CheckAccountExists(span.Context(), s.db, httpReq); err != nil {
		http.Error(w, "Account already exists", http.StatusConflict)
		return
	}

	account, err := repository.CreateAccount(span.Context(), s.db, httpReq)
	if err != nil {
		http.Error(w, "Failed to create account", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(account)
}

func (s *Server) handleCreateSession(w http.ResponseWriter, r *http.Request) {
	span := telemetry.SpanFromRequest(r)
	decoder := json.NewDecoder(r.Body)
	var httpReq model.CreateSessionRequest
	if err := decoder.Decode(&httpReq); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	account, err := repository.ValidateCredentials(span.Context(), s.db, httpReq)
	if err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	sessionID, err := repository.CreateSession(span.Context(), s.db, account.ID)
	if err != nil {
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	if s.sys.IsActorSpawned(r.Context(), account.ID) == system.ActorStateNotFound {
		if _, err := s.sys.AttemptRestoreActor(span.Context(), "AccountActor", account.ID); err != nil {
			slog.Warn("Failed to restore account actor", "error", err, "accountID", account.ID)
		}

		// Actor not found, spawn a new one
		if _, err := s.sys.SpawnWithParams(span.Context(), "AccountActor", actor.AccountActorParams{ID: account.ID, Name: account.Username}); err != nil {
			slog.Error("Failed to spawn account actor", "error", err, "accountID", account.ID)
			http.Error(w, "Failed to spawn account actor", http.StatusInternalServerError)
			return
		}
		slog.Info("Spawned new account actor for session", "accountID", account.ID, "sessionID", sessionID)
	} else {
		slog.Info("Account actor already spawned for session", "accountID", account.ID, "sessionID", sessionID)
	}

	token := paseto.CreateSessionToken(span.Context(), account.ID, sessionID)

	response := model.CreateSessionResponse{
		ID:    sessionID,
		Token: token,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleDeleteSession(w http.ResponseWriter, r *http.Request) {
	span := telemetry.SpanFromRequest(r)

	sessionID, err := paseto.ValidateSessionTokenFromRequest(r.Context(), r)
	if err != nil {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	if err := repository.ValidateSession(r.Context(), s.db, sessionID); err != nil {
		slog.Error("Invalid session", "error", err)
		http.Error(w, "Invalid session", http.StatusBadRequest)
		return
	}

	if err := repository.DeleteSession(span.Context(), s.db, sessionID); err != nil {
		http.Error(w, "Failed to delete session", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Session deleted successfully"))
}
