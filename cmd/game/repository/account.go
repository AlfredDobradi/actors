package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/alfreddobradi/actors/cmd/game/model"
	"github.com/alfreddobradi/actors/pkg/database"
	"github.com/alfreddobradi/actors/pkg/telemetry"
	"github.com/google/uuid"
)

func CheckAccountExists(ctx context.Context, db database.DB, req model.CreateAccountRequest) error {
	span := telemetry.SpanFromContext(ctx)
	span.GetLogger().Info("Checking if account exists", "username", req.Username, "email", req.Email)

	keys := db.Keys(ctx)
	for _, key := range keys {
		if !strings.HasPrefix(key, "account:") {
			continue
		}

		val, ok := db.Get(ctx, key, false)
		if !ok {
			continue
		}

		var account model.Account
		if err := json.Unmarshal([]byte(val[key]), &account); err != nil {
			continue
		}

		if account.Username == req.Username {
			return fmt.Errorf("username already exists")
		}

		if account.Email == req.Email {
			return fmt.Errorf("email already exists")
		}
	}

	return nil
}

func CreateAccount(ctx context.Context, db database.DB, req model.CreateAccountRequest) (model.CreateAccountResponse, error) {
	span := telemetry.SpanFromContext(ctx)
	span.GetLogger().Info("Creating account", "username", req.Username, "email", req.Email)

	account := model.Account{
		ID:        uuid.New(),
		Username:  req.Username,
		Email:     req.Email,
		Password:  req.Password, // TODO hash
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Active:    true,
	}

	if err := db.Set(ctx, "account:"+account.ID.String(), account); err != nil {
		return model.CreateAccountResponse{}, err
	}

	return model.CreateAccountResponse{ID: account.ID, Username: account.Username, Email: account.Email}, nil
}

func ValidateCredentials(ctx context.Context, db database.DB, req model.CreateSessionRequest) (model.Account, error) {
	span := telemetry.SpanFromContext(ctx)
	span.GetLogger().Info("Validating credentials", "username", req.Username)

	keys := db.Keys(ctx)
	for _, key := range keys {
		if !strings.HasPrefix(key, "account:") {
			continue
		}

		val, ok := db.Get(ctx, key, false)
		if !ok {
			continue
		}

		var account model.Account
		if err := json.Unmarshal([]byte(val[key]), &account); err != nil {
			continue
		}

		if account.Username == req.Username && account.Password == req.Password { // TODO hash
			return account, nil
		}
	}

	return model.Account{}, fmt.Errorf("invalid credentials")
}

func CreateSession(ctx context.Context, db database.DB, accountID uuid.UUID) (uuid.UUID, error) {
	span := telemetry.SpanFromContext(ctx)
	span.GetLogger().Info("Creating session", "account_id", accountID)

	session := model.Session{
		ID:        uuid.New(),
		AccountID: accountID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Active:    true,
	}

	if err := db.Set(ctx, "session:"+session.ID.String(), session); err != nil {
		return uuid.Nil, err
	}

	return session.ID, nil
}

func GetAccountBySessionID(ctx context.Context, db database.DB, sessionID uuid.UUID) (model.Account, error) {
	span := telemetry.SpanFromContext(ctx)
	span.GetLogger().Info("Getting account by session ID", "session_id", sessionID)

	sessionKey := "session:" + sessionID.String()
	sessionVal, ok := db.Get(ctx, sessionKey, false)
	if !ok {
		return model.Account{}, fmt.Errorf("session not found")
	}

	var session model.Session
	if err := json.Unmarshal([]byte(sessionVal[sessionKey]), &session); err != nil {
		return model.Account{}, fmt.Errorf("invalid session data")
	}

	accountKey := "account:" + session.AccountID.String()
	accountVal, ok := db.Get(ctx, accountKey, false)
	if !ok {
		return model.Account{}, fmt.Errorf("account not found")
	}

	var account model.Account
	if err := json.Unmarshal([]byte(accountVal[accountKey]), &account); err != nil {
		return model.Account{}, fmt.Errorf("invalid account data")
	}

	return account, nil
}

func ValidateSession(ctx context.Context, db database.DB, sessionID uuid.UUID) error {
	span := telemetry.SpanFromContext(ctx)
	span.GetLogger().Info("Validating session", "session_id", sessionID)

	sessionKey := "session:" + sessionID.String()
	sessionVal, ok := db.Get(ctx, sessionKey, false)
	if !ok {
		return fmt.Errorf("session not found")
	}

	var session model.Session
	if err := json.Unmarshal([]byte(sessionVal[sessionKey]), &session); err != nil {
		return fmt.Errorf("invalid session data")
	}

	if !session.Active {
		return fmt.Errorf("session has been revoked")
	}

	return nil
}

func DeleteSession(ctx context.Context, db database.DB, sessionID uuid.UUID) error {
	span := telemetry.SpanFromContext(ctx)
	span.GetLogger().Info("Deleting session", "session_id", sessionID)

	sessionKey := "session:" + sessionID.String()
	if err := db.Delete(ctx, sessionKey); err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	return nil
}

func GetAccounts(ctx context.Context, db database.DB) ([]model.Account, error) {
	span := telemetry.SpanFromContext(ctx)
	span.GetLogger().Info("Retrieving all accounts")

	accountVals, ok := db.Get(ctx, "account:", true)
	if !ok {
		return nil, fmt.Errorf("account not found")
	}

	accounts := make([]model.Account, 0, len(accountVals))
	for _, v := range accountVals {
		var acc model.Account
		if err := json.Unmarshal([]byte(v), &acc); err != nil {
			return nil, err
		}
		accounts = append(accounts, acc)
	}

	return accounts, nil
}

func GetAccount(ctx context.Context, db database.DB, accountID string) (model.Account, error) {
	span := telemetry.SpanFromContext(ctx)
	span.GetLogger().Info("Retrieving all accounts")

	key := fmt.Sprintf("account:%s", accountID)

	accountVal, ok := db.Get(ctx, key, false)
	if !ok {
		return model.Account{}, fmt.Errorf("account not found")
	}

	var acc model.Account
	if err := json.Unmarshal([]byte(accountVal[key]), &acc); err != nil {
		return model.Account{}, err
	}

	return acc, nil
}
