package mock

import (
	"context"
	"fmt"
	"time"

	"github.com/alfreddobradi/actors/cmd/game/game"
	"github.com/alfreddobradi/actors/cmd/game/model"
	"github.com/alfreddobradi/actors/pkg/telemetry"
	"github.com/google/uuid"
)

var (
	ErrAccountNotFound      = fmt.Errorf("account not found")
	ErrSessionNotFound      = fmt.Errorf("session not found")
	ErrSessionRevoked       = fmt.Errorf("session has been revoked")
	ErrAccountActorNotFound = fmt.Errorf("account actor not found")
	ErrGuildConfigNotFound  = fmt.Errorf("guild config not found")
)

func (r *Repository) CheckAccountExists(ctx context.Context, req model.CreateAccountRequest) error {
	span := telemetry.SpanFromContext(ctx)
	span.GetLogger().Info("Checking if account exists", "username", req.Username, "email", req.Email)

	for _, account := range r.accounts {
		if account.Email == req.Email {
			return fmt.Errorf("email already exists")
		}

		if account.Username == req.Username {
			return fmt.Errorf("username already exists")
		}
	}

	return nil
}

func (r *Repository) CreateAccount(ctx context.Context, req model.CreateAccountRequest) (model.CreateAccountResponse, error) {
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

	if existsErr := r.CheckAccountExists(ctx, req); existsErr != nil {
		return model.CreateAccountResponse{}, existsErr
	}

	r.accounts[account.ID] = account

	return model.CreateAccountResponse{ID: account.ID, Username: account.Username, Email: account.Email}, nil
}

func (r *Repository) ValidateCredentials(ctx context.Context, req model.CreateSessionRequest) (model.Account, error) {
	span := telemetry.SpanFromContext(ctx)
	span.GetLogger().Info("Validating credentials", "username", req.Username)

	for _, account := range r.accounts {
		if account.Username == req.Username && account.Password == req.Password {
			return account, nil
		}
	}

	return model.Account{}, ErrAccountNotFound
}

func (r *Repository) CreateSession(ctx context.Context, accountID uuid.UUID) (uuid.UUID, error) {
	span := telemetry.SpanFromContext(ctx)
	span.GetLogger().Info("Creating session", "account_id", accountID)

	session := model.Session{
		ID:        uuid.New(),
		AccountID: accountID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Active:    true,
	}

	r.sessions[session.ID] = session

	return session.ID, nil
}

func (r *Repository) GetAccountBySessionID(ctx context.Context, sessionID uuid.UUID) (model.Account, error) {
	span := telemetry.SpanFromContext(ctx)
	span.GetLogger().Info("Getting account by session ID", "session_id", sessionID)

	session, ok := r.sessions[sessionID]
	if !ok {
		return model.Account{}, ErrSessionNotFound
	}

	account, ok := r.accounts[session.AccountID]
	if !ok {
		return model.Account{}, ErrAccountNotFound
	}

	return account, nil
}

func (r *Repository) ValidateSession(ctx context.Context, sessionID uuid.UUID) error {
	span := telemetry.SpanFromContext(ctx)
	span.GetLogger().Info("Validating session", "session_id", sessionID)

	session, ok := r.sessions[sessionID]
	if !ok {
		return ErrSessionNotFound
	}

	if !session.Active {
		return ErrSessionRevoked
	}

	return nil
}

func (r *Repository) DeleteSession(ctx context.Context, sessionID uuid.UUID) error {
	span := telemetry.SpanFromContext(ctx)
	span.GetLogger().Info("Deleting session", "session_id", sessionID)

	delete(r.sessions, sessionID)

	return nil
}

func (r *Repository) RevokeSession(ctx context.Context, sessionID uuid.UUID) error {
	span := telemetry.SpanFromContext(ctx)
	span.GetLogger().Info("Revoking session", "session_id", sessionID)

	session, ok := r.sessions[sessionID]
	if !ok {
		return ErrSessionNotFound
	}

	session.Active = false

	return nil
}

func (r *Repository) GetAccounts(ctx context.Context) ([]model.Account, error) {
	span := telemetry.SpanFromContext(ctx)
	span.GetLogger().Info("Retrieving all accounts")

	accounts := make([]model.Account, 0)
	for _, account := range r.accounts {
		accounts = append(accounts, account)
	}

	return accounts, nil
}

func (r *Repository) GetAccount(ctx context.Context, accountID string) (model.Account, error) {
	span := telemetry.SpanFromContext(ctx)
	span.GetLogger().Info("Retrieving account", "account_id", accountID)

	id, err := uuid.Parse(accountID)
	if err != nil {
		return model.Account{}, err
	}

	account, ok := r.accounts[id]
	if !ok {
		return model.Account{}, ErrAccountNotFound
	}

	return account, nil
}

func (r *Repository) GetSessionsByAccountID(ctx context.Context, accountID string) ([]model.Session, error) {
	span := telemetry.SpanFromContext(ctx)
	span.GetLogger().Info("Retrieving sessions for account", "account_id", accountID)

	id, err := uuid.Parse(accountID)
	if err != nil {
		return nil, err
	}

	sessions := make([]model.Session, 0)
	for _, session := range r.sessions {
		if session.AccountID == id {
			sessions = append(sessions, session)
		}
	}

	return sessions, nil
}

func (r *Repository) GetGuildConfig(ctx context.Context, accountID uuid.UUID) (*game.GuildConfig, error) {
	span := telemetry.SpanFromContext(ctx)
	span.GetLogger().Info("Retrieving guild config for account", "account_id", accountID)

	c, ok := r.guildConfigs[accountID]
	if !ok {
		return nil, ErrGuildConfigNotFound
	}

	return &c, nil
}

func (r *Repository) PersistAccountActor(ctx context.Context, account model.AccountActor) error {
	span := telemetry.SpanFromContext(ctx)
	span.GetLogger().Info("Persisting account data", "account_id", account.ID)

	r.accountActors[account.ID] = account

	return nil
}

func (r *Repository) RestoreAccountActor(ctx context.Context, accountID uuid.UUID) (*model.AccountActor, error) {
	span := telemetry.SpanFromContext(ctx)
	span.GetLogger().Info("Restoring account data", "account_id", accountID)

	accountData, ok := r.accountActors[accountID]
	if !ok {
		return nil, ErrAccountActorNotFound
	}

	return &accountData, nil
}
