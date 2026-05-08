package middleware

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/alfreddobradi/actors/examples/game/model"
	"github.com/alfreddobradi/actors/examples/game/paseto"
	"github.com/alfreddobradi/actors/examples/game/repository"
	"github.com/alfreddobradi/actors/pkg/database"
)

func Authorization(db database.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sessionID, err := paseto.ValidateSessionTokenFromRequest(r.Context(), r)
			if err != nil {
				slog.Error("Invalid token", "error", err)
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}

			if err := repository.ValidateSession(r.Context(), db, sessionID); err != nil {
				slog.Error("Invalid session", "error", err)
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}

			accountData, err := repository.GetAccountBySessionID(r.Context(), db, sessionID)
			if err != nil {
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), model.ContextKeyAccountData, accountData)
			ctx = context.WithValue(ctx, model.ContextKeySessionID, sessionID)
			r = r.WithContext(ctx)

			next.ServeHTTP(w, r)
		})
	}
}
