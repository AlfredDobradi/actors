package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/alfreddobradi/actors/cmd/game/game"
	"github.com/alfreddobradi/actors/cmd/game/model"
	"github.com/alfreddobradi/actors/pkg/database"
	"github.com/alfreddobradi/actors/pkg/database/postgres"
	"github.com/alfreddobradi/actors/pkg/telemetry"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

func CheckAccountExistsKV(ctx context.Context, db database.KeyValue, req model.CreateAccountRequest) error {
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

func CheckAccountExists(ctx context.Context, db *postgres.Connection, req model.CreateAccountRequest) error {
	span := telemetry.SpanFromContext(ctx)
	span.GetLogger().Info("Checking if account exists", "username", req.Username, "email", req.Email)

	type Res struct {
		ID       uuid.UUID `db:"id"`
		Username string    `db:"username"`
		Email    string    `db:"email"`
	}

	var row Res
	err := db.Get(&row, "SELECT id, username, email FROM accounts WHERE username = $1 OR email = $2", req.Username, req.Email)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return nil
		}
		return err
	}

	if row.Email == req.Email {
		return fmt.Errorf("email already exists")
	}

	if row.Username == req.Username {
		return fmt.Errorf("username already exists")
	}

	return nil
}

func CreateAccountKV(ctx context.Context, db database.KeyValue, req model.CreateAccountRequest) (model.CreateAccountResponse, error) {
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

func CreateAccount(ctx context.Context, db *postgres.Connection, req model.CreateAccountRequest) (model.CreateAccountResponse, error) {
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

	if _, err := db.NamedExec("INSERT INTO accounts (id, username, email, password, created_at, updated_at, active) VALUES (:id, :username, :email, :password, :created_at, :updated_at, :active)", account); err != nil {
		return model.CreateAccountResponse{}, err
	}

	return model.CreateAccountResponse{ID: account.ID, Username: account.Username, Email: account.Email}, nil
}

func ValidateCredentialsKV(ctx context.Context, db database.KeyValue, req model.CreateSessionRequest) (model.Account, error) {
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

func ValidateCredentials(ctx context.Context, db *postgres.Connection, req model.CreateSessionRequest) (model.Account, error) {
	span := telemetry.SpanFromContext(ctx)
	span.GetLogger().Info("Validating credentials", "username", req.Username)

	var acc model.Account
	if err := db.Get(&acc, "SELECT * FROM accounts WHERE username = $1 AND password = $2", req.Username, req.Password); err != nil {
		return model.Account{}, err
	}

	return acc, nil
}

func CreateSessionKV(ctx context.Context, db database.KeyValue, accountID uuid.UUID) (uuid.UUID, error) {
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

func CreateSession(ctx context.Context, db *postgres.Connection, accountID uuid.UUID) (uuid.UUID, error) {
	span := telemetry.SpanFromContext(ctx)
	span.GetLogger().Info("Creating session", "account_id", accountID)

	session := model.Session{
		ID:        uuid.New(),
		AccountID: accountID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Active:    true,
	}

	if _, err := db.NamedExec("INSERT INTO sessions (id, account_id, created_at, updated_at, active) VALUES (:id, :account_id, :created_at, :updated_at, :active)", session); err != nil {
		return uuid.Nil, err
	}

	return session.ID, nil
}

func GetAccountBySessionIDKV(ctx context.Context, db database.KeyValue, sessionID uuid.UUID) (model.Account, error) {
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

func GetAccountBySessionID(ctx context.Context, db *postgres.Connection, sessionID uuid.UUID) (model.Account, error) {
	span := telemetry.SpanFromContext(ctx)
	span.GetLogger().Info("Getting account by session ID", "session_id", sessionID)

	type Res struct {
		AccountID uuid.UUID `db:"account_id"`
	}
	var sess Res
	if err := db.Get(&sess, "SELECT account_id FROM sessions WHERE id = $1", sessionID); err != nil {
		return model.Account{}, err
	}

	var acc model.Account
	if err := db.Get(&acc, "SELECT * FROM accounts WHERE id = $1", sess.AccountID); err != nil {
		return model.Account{}, err
	}

	return acc, nil
}

func ValidateSessionKV(ctx context.Context, db database.KeyValue, sessionID uuid.UUID) error {
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

func ValidateSession(ctx context.Context, db *postgres.Connection, sessionID uuid.UUID) error {
	span := telemetry.SpanFromContext(ctx)
	span.GetLogger().Info("Validating session", "session_id", sessionID)

	var session model.Session
	if err := db.Get(&session, "SELECT * FROM sessions WHERE id = $1", sessionID); err != nil {
		return err
	}

	if !session.Active {
		return fmt.Errorf("session has been revoked")
	}

	return nil
}

func DeleteSessionKV(ctx context.Context, db database.KeyValue, sessionID uuid.UUID) error {
	span := telemetry.SpanFromContext(ctx)
	span.GetLogger().Info("Deleting session", "session_id", sessionID)

	sessionKey := "session:" + sessionID.String()
	if err := db.Delete(ctx, sessionKey); err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	return nil
}

func DeleteSession(ctx context.Context, db *postgres.Connection, sessionID uuid.UUID) error {
	span := telemetry.SpanFromContext(ctx)
	span.GetLogger().Info("Deleting session", "session_id", sessionID)

	if _, err := db.Exec("DELETE FROM sessions WHERE id = $1", sessionID); err != nil {
		return err
	}

	return nil
}

func RevokeSession(ctx context.Context, db *postgres.Connection, sessionID uuid.UUID) error {
	span := telemetry.SpanFromContext(ctx)
	span.GetLogger().Info("Revoking session", "session_id", sessionID)

	if _, err := db.Exec("UPDATE sessions SET active = false WHERE id = $1", sessionID); err != nil {
		return err
	}

	return nil
}

func GetAccountsKV(ctx context.Context, db database.KeyValue) ([]model.Account, error) {
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

func GetAccounts(ctx context.Context, db *postgres.Connection) ([]model.Account, error) {
	span := telemetry.SpanFromContext(ctx)
	span.GetLogger().Info("Retrieving all accounts")

	accounts := make([]model.Account, 0)
	if err := db.Select(&accounts, "SELECT * FROM accounts"); err != nil {
		return nil, err
	}

	return accounts, nil
}

func GetAccountKV(ctx context.Context, db database.KeyValue, accountID string) (model.Account, error) {
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

func GetAccount(ctx context.Context, db *postgres.Connection, accountID string) (model.Account, error) {
	span := telemetry.SpanFromContext(ctx)
	span.GetLogger().Info("Retrieving all accounts")

	var account model.Account
	if err := db.Get(&account, "SELECT * FROM accounts WHERE id = $1", accountID); err != nil {
		return model.Account{}, err
	}

	return account, nil
}

func GetSessionsByAccountIDKV(ctx context.Context, db database.KeyValue, accountID string) ([]model.Session, error) {
	span := telemetry.SpanFromContext(ctx)
	span.GetLogger().Info("Retrieving sessions for account", "account_id", accountID)

	response := make([]model.Session, 0)

	sessionsVal, ok := db.Get(ctx, "session:", true)
	if !ok {
		return response, nil
	}

	for _, sessionRaw := range sessionsVal {
		var session model.Session
		if err := json.Unmarshal([]byte(sessionRaw), &session); err != nil {
			return make([]model.Session, 0), err
		}

		response = append(response, session)
	}

	return response, nil
}

func GetSessionsByAccountID(ctx context.Context, db *postgres.Connection, accountID string) ([]model.Session, error) {
	span := telemetry.SpanFromContext(ctx)
	span.GetLogger().Info("Retrieving sessions for account", "account_id", accountID)

	sessions := make([]model.Session, 0)
	if err := db.Select(&sessions, "SELECT * FROM sessions WHERE account_id = $1", accountID); err != nil {
		return nil, err
	}

	return sessions, nil
}

func PersistAccountActor(ctx context.Context, db *postgres.Connection, account model.AccountActor) error {
	span := telemetry.SpanFromContext(ctx)
	span.GetLogger().Info("Persisting account data", "account_id", account.ID)

	tx, err := db.Beginx()
	if err != nil {
		return err
	}

	if err := persistGuildData(span.Context(), tx, account.ID, account.Guild); err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			slog.Error("failed to roll back account data persistence", "error", rollbackErr)
		}
		return err
	}

	return tx.Commit()
}

func persistGuildData(ctx context.Context, tx *sqlx.Tx, accountID uuid.UUID, guild *game.Guild) error {
	span := telemetry.SpanFromContext(ctx)
	span.GetLogger().Info("Persisting guild data", "account_id", accountID)

	// upsert guild data (account_id, name, gold)
	if _, err := tx.Exec("INSERT INTO guilds (id, account_id, name, gold, updated_at) VALUES ($1, $2, $3, $4, $5) ON CONFLICT (id, account_id) DO UPDATE SET gold = $4, updated_at = $5",
		guild.ID(),
		accountID,
		guild.Name(),
		guild.Gold.Load(),
		time.Now(),
	); err != nil {
		return err
	}

	for _, hero := range guild.Heroes() {
		action := make(map[string]any)
		if hero.Action != nil {
			rawAction, err := json.Marshal(hero.Action)
			if err != nil {
				return err
			}
			var actionData map[string]any
			if err := json.Unmarshal(rawAction, &actionData); err != nil {
				return err
			}
			action["name"] = hero.Action.GetName()
			action["data"] = actionData
		}

		actionBytes, err := json.Marshal(action)
		if err != nil {
			return err
		}

		if _, err := tx.Exec("INSERT INTO heroes (id, guild_id, name, level, experience, status, cooldown, health, energy, gold, action) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11) ON CONFLICT (id) DO UPDATE SET level = $4, experience = $5, status = $6, cooldown = $7, health = $8, energy = $9, gold = $10, action = $11",
			hero.ID,
			guild.ID(),
			hero.Name,
			hero.Level,
			hero.Experience,
			hero.Status,
			hero.Cooldown,
			hero.Health,
			hero.Energy,
			hero.Gold,
			string(actionBytes),
		); err != nil {
			return err
		}

		for resource, amount := range hero.Inventory.Resources() {
			if _, err := tx.Exec("INSERT INTO hero_resources (hero_id, name, amount) VALUES ($1, $2, $3) ON CONFLICT (hero_id, name) DO UPDATE SET amount = $3",
				hero.ID,
				resource,
				amount,
			); err != nil {
				return err
			}
		}
	}

	return nil
}
