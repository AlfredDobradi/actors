package repository

import (
	"context"
	"testing"
	"time"

	"github.com/alfreddobradi/actors/examples/game/database/mmap"
	"github.com/alfreddobradi/actors/examples/game/model"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestCreateAccount(t *testing.T) {
	db, err := mmap.NewStore("")
	require.NoError(t, err)

	req := model.CreateAccountRequest{
		Username: "testuser",
		Email:    "testuser@example.com",
		Password: "password123",
	}

	resp, err := CreateAccount(context.Background(), db, req)
	require.NoError(t, err)
	require.Equal(t, req.Username, resp.Username)
	require.Equal(t, req.Email, resp.Email)

	// Verify account is stored in the database
	storedAccount, exists := db.Get("account:" + resp.ID.String())
	require.True(t, exists)
	require.NotNil(t, storedAccount)
}

func TestValidateCredentials(t *testing.T) {
	db, err := mmap.NewStore("")
	require.NoError(t, err)

	// Create a test account
	account := model.Account{
		ID:        uuid.New(),
		Username:  "testuser",
		Email:     "testuser@example.com",
		Password:  "password123",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Active:    true,
	}

	// Store the test account in the database
	err = db.Set("account:"+account.ID.String(), account)
	require.NoError(t, err)

	// Validate credentials
	req := model.CreateSessionRequest{
		Username: "testuser",
		Password: "password123",
	}

	validatedAccount, err := ValidateCredentials(context.Background(), db, req)
	require.NoError(t, err)
	require.Equal(t, account.ID, validatedAccount.ID)
	require.Equal(t, account.Username, validatedAccount.Username)
	require.Equal(t, account.Email, validatedAccount.Email)
}
