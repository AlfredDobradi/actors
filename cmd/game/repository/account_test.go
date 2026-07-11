package repository

import (
	"context"
	"testing"
	"time"

	"github.com/alfreddobradi/actors/cmd/game/model"
	"github.com/alfreddobradi/actors/pkg/database/memory"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

const (
	testUser     = "testuser"
	testEmail    = "testuser@example.com"
	testPassword = "password123"
)

func TestCreateAccount(t *testing.T) {
	db := memory.NewStore()

	req := model.CreateAccountRequest{
		Username: testUser,
		Email:    testEmail,
		Password: testPassword,
	}

	resp, err := CreateAccount(context.Background(), db, req)
	require.NoError(t, err)
	require.Equal(t, req.Username, resp.Username)
	require.Equal(t, req.Email, resp.Email)

	// Verify account is stored in the database
	storedAccount, exists := db.Get(context.Background(), "account:"+resp.ID.String(), false)
	require.True(t, exists)
	require.NotNil(t, storedAccount)
}

func TestValidateCredentials(t *testing.T) {
	db := memory.NewStore()

	// Create a test account
	account := model.Account{
		ID:        uuid.New(),
		Username:  testUser,
		Email:     testEmail,
		Password:  testPassword,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Active:    true,
	}

	// Store the test account in the database
	err := db.Set(context.Background(), "account:"+account.ID.String(), account)
	require.NoError(t, err)

	// Validate credentials
	req := model.CreateSessionRequest{
		Username: testUser,
		Password: testPassword,
	}

	validatedAccount, err := ValidateCredentials(context.Background(), db, req)
	require.NoError(t, err)
	require.Equal(t, account.ID, validatedAccount.ID)
	require.Equal(t, account.Username, validatedAccount.Username)
	require.Equal(t, account.Email, validatedAccount.Email)
}
