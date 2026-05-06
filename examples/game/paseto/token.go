package paseto

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"aidanwoods.dev/go-paseto"
	"github.com/google/uuid"
)

var symmetricKey *paseto.V4SymmetricKey

func readKeyFromEnv() (*paseto.V4SymmetricKey, error) {
	rawKey := os.Getenv("PASETO_KEY")

	if rawKey == "" {
		return nil, os.ErrNotExist
	}

	key, err := paseto.V4SymmetricKeyFromHex(rawKey)
	if err != nil {
		return nil, err
	}

	return &key, nil
}

func GetKey() paseto.V4SymmetricKey {
	if symmetricKey == nil {
		key, err := readKeyFromEnv()
		if err == nil {
			symmetricKey = key
			slog.Info("Loaded symmetric key for PASETO encryption from environment variable", "key", key.ExportHex())
		} else {
			slog.Warn("PASETO_KEY environment variable is not set or invalid, falling back to random key")

			newKey := paseto.NewV4SymmetricKey()
			slog.Info("Generated random symmetric key for PASETO encryption, set PASETO_KEY environment variable to persist across restarts", "key", newKey.ExportHex())

			symmetricKey = &newKey
		}
	}

	return *symmetricKey
}

func CreateSessionToken(ctx context.Context, accountID uuid.UUID, sessionID uuid.UUID) string {
	key := GetKey()

	token := paseto.NewToken()
	token.SetIssuedAt(time.Now())
	token.SetNotBefore(time.Now())
	token.SetExpiration(time.Now().Add(24 * time.Hour))
	token.SetSubject(accountID.String())
	token.SetIssuer("some-game")

	token.SetString("session_id", sessionID.String())

	return token.V4Encrypt(key, nil)
}

func ValidateSessionToken(tokenStr string) (uuid.UUID, error) {
	key := GetKey()

	parser := paseto.NewParser()
	parser.AddRule(paseto.IssuedBy("some-game"))
	parser.AddRule(paseto.NotBeforeNbf())
	parser.AddRule(paseto.NotExpired())

	token, err := parser.ParseV4Local(key, tokenStr, nil)
	if err != nil {
		return uuid.Nil, err
	}

	sessionIDStr, err := token.GetString("session_id")
	if err != nil {
		return uuid.Nil, err
	}

	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		return uuid.Nil, err
	}

	return sessionID, nil
}

func ValidateSessionTokenFromRequest(ctx context.Context, r *http.Request) (uuid.UUID, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return uuid.Nil, fmt.Errorf("missing Authorization header")
	}

	tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
	return ValidateSessionToken(tokenStr)
}
