package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/alfreddobradi/actors/cmd/game/api/state"
	"github.com/alfreddobradi/actors/cmd/game/model"
	"github.com/alfreddobradi/actors/cmd/game/paseto"
	"github.com/alfreddobradi/actors/cmd/game/repository"
	sysmodel "github.com/alfreddobradi/actors/pkg/model"
	"github.com/alfreddobradi/actors/pkg/system"
	"github.com/alfreddobradi/actors/pkg/telemetry"
)

func HandleCreateAccount(s *state.Context) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		span := telemetry.SpanFromRequest(r)

		repo, repoErr := repository.Get(s.DB)
		if repoErr != nil {
			slog.Error("Failed to get repository", "error", repoErr)
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		decoder := json.NewDecoder(r.Body)
		var httpReq model.CreateAccountRequest
		if err := decoder.Decode(&httpReq); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		defer closeBody(r.Body)

		if err := repo.CheckAccountExists(span.Context(), httpReq); err != nil {
			slog.Error("account already exists", "error", err.Error())
			http.Error(w, "Account already exists", http.StatusConflict)
			return
		}

		account, err := repo.CreateAccount(span.Context(), httpReq)
		if err != nil {
			http.Error(w, "Failed to create account", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(account); err != nil {
			slog.Warn("Failed to write response", "error", err)
		}
	}
}

func HandleCreateSession(s *state.Context) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		span := telemetry.SpanFromRequest(r)

		repo, repoErr := repository.Get(s.DB)
		if repoErr != nil {
			slog.Error("Failed to get repository", "error", repoErr)
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		decoder := json.NewDecoder(r.Body)
		var httpReq model.CreateSessionRequest
		if err := decoder.Decode(&httpReq); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		defer closeBody(r.Body)

		account, err := repo.ValidateCredentials(span.Context(), httpReq)
		if err != nil {
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			return
		}

		sessionID, err := repo.CreateSession(span.Context(), account.ID)
		if err != nil {
			http.Error(w, "Failed to create session", http.StatusInternalServerError)
			return
		}

		if s.System.IsActorSpawned(r.Context(), account.ID) == sysmodel.ActorStateNotFound {
			if _, err := s.System.AttemptRestoreActor(span.Context(), "AccountActor", model.AccountActorParams{ID: account.ID}, system.WithSubscription("tick")); err != nil {
				slog.Warn("Failed to restore account actor", "error", err, "accountID", account.ID)
			}

			// Actor not found, spawn a new one
			if _, err := s.System.SpawnWithParams(span.Context(), "AccountActor", model.AccountActorParams{ID: account.ID, Name: account.Username}, system.WithSubscription("tick")); err != nil {
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
		if err := json.NewEncoder(w).Encode(response); err != nil {
			slog.Warn("Failed to write response", "error", err)
		}
	}
}

func HandleDeleteSession(s *state.Context) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		span := telemetry.SpanFromRequest(r)

		repo, repoErr := repository.Get(s.DB)
		if repoErr != nil {
			slog.Error("Failed to get repository", "error", repoErr)
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		sessionID, err := paseto.ValidateSessionTokenFromRequest(r.Context(), r)
		if err != nil {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		if err := repo.ValidateSession(span.Context(), sessionID); err != nil {
			slog.Error("Invalid session", "error", err)
			http.Error(w, "Invalid session", http.StatusBadRequest)
			return
		}

		if err := repo.DeleteSession(span.Context(), sessionID); err != nil {
			http.Error(w, "Failed to delete session", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("Session deleted successfully")); err != nil {
			slog.Warn("Failed to write response", "error", err)
		}
	}
}
