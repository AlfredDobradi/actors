package middleware

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/alfreddobradi/actors/cmd/game/model"
	"github.com/alfreddobradi/actors/cmd/game/paseto"
	"github.com/alfreddobradi/actors/cmd/game/repository"
	"github.com/alfreddobradi/actors/pkg/database"
)

func Authorization(db database.Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sessionID, err := paseto.ValidateSessionTokenFromRequest(r.Context(), r)
			if err != nil {
				slog.Error("Invalid token", "error", err)
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}

			repo, repoErr := repository.Get(db)
			if repoErr != nil {
				slog.Error("Failed to get repository", "error", repoErr)
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}

			if validationErr := repo.ValidateSession(r.Context(), sessionID); validationErr != nil {
				slog.Error("Invalid session", "error", validationErr)
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}

			accountData, err := repo.GetAccountBySessionID(r.Context(), sessionID)
			if err != nil {
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), model.ContextKeyAccountData, &accountData)
			ctx = context.WithValue(ctx, model.ContextKeySessionID, sessionID)
			r = r.WithContext(ctx)

			next.ServeHTTP(w, r)
		})
	}
}
