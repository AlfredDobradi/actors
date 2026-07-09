package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/alfreddobradi/actors/cmd/game/game"
	"github.com/alfreddobradi/actors/cmd/game/model"
	"github.com/alfreddobradi/actors/pkg/database/postgres"
	"github.com/alfreddobradi/actors/pkg/telemetry"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

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

	type Aux struct {
		Username string `db:"username"`
		Email    string `db:"email"`
	}

	c := make([]Aux, 0)
	if err := db.Select(&c, "SELECT username, email FROM accounts WHERE username = $1 OR email = $2", req.Username, req.Email); err != nil {
		if err != sql.ErrNoRows {
			return model.CreateAccountResponse{}, err
		}
	}

	if len(c) > 0 {
		return model.CreateAccountResponse{}, fmt.Errorf("this username or email has already been used")
	}

	if _, err := db.NamedExec("INSERT INTO accounts (id, username, email, password, created_at, updated_at, active) VALUES (:id, :username, :email, :password, :created_at, :updated_at, :active)", account); err != nil {
		return model.CreateAccountResponse{}, err
	}

	return model.CreateAccountResponse{ID: account.ID, Username: account.Username, Email: account.Email}, nil
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

func GetAccounts(ctx context.Context, db *postgres.Connection) ([]model.Account, error) {
	span := telemetry.SpanFromContext(ctx)
	span.GetLogger().Info("Retrieving all accounts")

	accounts := make([]model.Account, 0)
	if err := db.Select(&accounts, "SELECT * FROM accounts"); err != nil {
		return nil, err
	}

	return accounts, nil
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

	if account.Guild != nil {
		if err := persistGuildData(span.Context(), tx, account.ID, account.Guild); err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				slog.Error("failed to roll back account data persistence", "error", rollbackErr)
			}
			return err
		}
	}

	return tx.Commit()
}

type actionAux struct {
	Name string         `json:"name"`
	Data map[string]any `json:"data"`
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
		action := actionAux{}
		if hero.Action != nil {
			rawAction, err := json.Marshal(hero.Action)
			if err != nil {
				return err
			}
			var actionData map[string]any
			if err := json.Unmarshal(rawAction, &actionData); err != nil {
				return err
			}
			action.Name = hero.Action.GetName()
			action.Data = actionData
		}

		actionBytes, err := json.Marshal(action)
		if err != nil {
			return err
		}

		if _, err := tx.Exec("INSERT INTO heroes (id, guild_id, name, level, experience, status, cooldown, health, energy, gold, last_tick, action) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12) ON CONFLICT (id) DO UPDATE SET level = $4, experience = $5, status = $6, cooldown = $7, health = $8, energy = $9, gold = $10, last_tick = $11, action = $12",
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
			hero.LastTick,
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

func RestoreAccountActor(ctx context.Context, db *postgres.Connection, accountID uuid.UUID) (*model.AccountActor, error) {
	span := telemetry.SpanFromContext(ctx)
	span.GetLogger().Info("Restoring account data", "account_id", accountID)

	accountData, err := restoreAccountData(ctx, db, accountID)
	if err != nil {
		return nil, err
	}

	guildData, err := restoreGuildData(span.Context(), db, accountID)
	if err != nil {
		return nil, err
	}

	accountData.Guild = guildData

	return accountData, nil
}

func restoreGuildData(ctx context.Context, db *postgres.Connection, accountID uuid.UUID) (*game.Guild, error) {
	guildAux := game.GuildAux{}
	if err := db.Get(&guildAux, "SELECT id, name, gold FROM guilds WHERE account_id = $1", accountID); err != nil {
		return nil, err
	}

	guild := game.GuildFromAux(guildAux)

	heroes := make([]game.Hero, 0)

	if err := db.Select(&heroes, "SELECT id, name, guild_id, level, experience, status, cooldown, health, energy, gold, last_tick FROM heroes WHERE guild_id = $1", guild.ID()); err != nil {
		return nil, err
	}

	var heroError error
	for _, hero := range heroes {
		hero.Inventory, heroError = restoreHeroInventory(ctx, db, hero.ID)
		if heroError != nil {
			return nil, heroError
		}

		hero.Action, heroError = restoreHeroAction(ctx, db, hero.ID)
		if heroError != nil {
			return nil, heroError
		}

		guild.AddHero(&hero)
	}

	return guild, nil
}

type resourceAux struct {
	Name   string `db:"name"`
	Amount int    `db:"amount"`
}

func restoreHeroInventory(ctx context.Context, db *postgres.Connection, heroID uuid.UUID) (*game.Inventory, error) {
	invData := make([]resourceAux, 0)
	if err := db.Select(&invData, "SELECT name, amount FROM hero_resources WHERE hero_id = $1", heroID); err != nil {
		return nil, err
	}

	inventory := game.NewInventory()

	for _, resource := range invData {
		res, ok := game.ResourceByName(resource.Name)
		if !ok {
			slog.Warn("invalid resource name", "name", resource.Name)
			continue
		}
		inventory.AddResource(res, resource.Amount)
	}

	return inventory, nil
}

func restoreHeroAction(ctx context.Context, db *postgres.Connection, heroID uuid.UUID) (game.Action, error) {
	actionRaw := []byte("")
	if err := db.Get(&actionRaw, "SELECT action FROM heroes WHERE id = $1", heroID); err != nil {
		return nil, err
	}

	action := actionAux{}
	if err := json.Unmarshal(actionRaw, &action); err != nil {
		return nil, err
	}

	a, ok := game.ActionMap[action.Name]
	if !ok {
		return nil, fmt.Errorf("failed to retrieve action for hero %s raw=%s", heroID, string(actionRaw))
	}

	switch action.Name {
	case game.ActionNameGather:
		resourceName := action.Data["Resource"].(map[string]any)["Name"].(string)
		resource, ok := game.ResourceByName(resourceName)
		if !ok {
			return nil, fmt.Errorf("invalid resource %s", resourceName)
		}

		aa := a.(*game.GatherAction)
		aa.Resource = resource
		a = aa
	}

	return a, nil

}

func restoreAccountData(ctx context.Context, db *postgres.Connection, accountID uuid.UUID) (*model.AccountActor, error) {
	account := model.AccountActor{}
	if err := db.Get(&account, "SELECT id, username as name FROM accounts WHERE id = $1", accountID); err != nil {
		return nil, err
	}

	return &account, nil
}
