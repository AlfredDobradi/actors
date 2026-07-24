package repository

import (
	"context"
	"fmt"

	"github.com/alfreddobradi/actors/cmd/game/model"
	pgrepo "github.com/alfreddobradi/actors/cmd/game/repository/postgres"
	"github.com/alfreddobradi/actors/pkg/database"
	"github.com/alfreddobradi/actors/pkg/database/store/postgres"
	"github.com/google/uuid"
)

type Repository interface {
	CheckAccountExists(ctx context.Context, req model.CreateAccountRequest) error
	CreateAccount(ctx context.Context, req model.CreateAccountRequest) (model.CreateAccountResponse, error)
	ValidateCredentials(ctx context.Context, req model.CreateSessionRequest) (model.Account, error)
	CreateSession(ctx context.Context, accountID uuid.UUID) (uuid.UUID, error)
	GetAccountBySessionID(ctx context.Context, sessionID uuid.UUID) (model.Account, error)
	ValidateSession(ctx context.Context, sessionID uuid.UUID) error
	DeleteSession(ctx context.Context, sessionID uuid.UUID) error
	RevokeSession(ctx context.Context, sessionID uuid.UUID) error
	GetAccounts(ctx context.Context) ([]model.Account, error)
	GetAccount(ctx context.Context, accountID string) (model.Account, error)
	GetSessionsByAccountID(ctx context.Context, accountID string) ([]model.Session, error)
	PersistAccountActor(ctx context.Context, account model.AccountActor) error
	RestoreAccountActor(ctx context.Context, accountID uuid.UUID) (*model.AccountActor, error)
}

func Get(db database.Store) (Repository, error) {
	switch d := db.(type) {
	case *postgres.Connection:
		return pgrepo.New(d), nil
	}
	return nil, fmt.Errorf("invalid database type: %T", db)
}
