package repository

import (
	"context"
	"testing"

	"github.com/alfreddobradi/actors/cmd/game/game"
	"github.com/alfreddobradi/actors/cmd/game/model"
	"github.com/alfreddobradi/actors/cmd/game/repository/mock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

const (
	testUser     string = "testUser"
	testPassword string = "password123"
)

func TestCreateAccount(t *testing.T) {
	req := model.CreateAccountRequest{
		Username: testUser,
		Email:    "testuser@example.com",
		Password: testPassword,
	}

	repo := mock.New()

	resp, err := repo.CreateAccount(context.Background(), req)
	require.NoError(t, err)
	require.Equal(t, req.Username, resp.Username)
	require.Equal(t, req.Email, resp.Email)

	// Verify account is stored in the database
	storedAccount, storeErr := repo.GetAccount(context.Background(), resp.ID.String())
	require.NoError(t, storeErr)
	require.Equal(t, req.Username, storedAccount.Username)
	require.Equal(t, req.Email, storedAccount.Email)
}

func TestValidateCredentials(t *testing.T) {
	repo := mock.New()

	createReq := model.CreateAccountRequest{
		Username: testUser,
		Email:    "testuser@example.com",
		Password: testPassword,
	}

	account, err := repo.CreateAccount(context.Background(), createReq)
	require.NoError(t, err)

	// Validate credentials
	validateReq := model.CreateSessionRequest{
		Username: testUser,
		Password: testPassword,
	}

	validatedAccount, err := repo.ValidateCredentials(context.Background(), validateReq)
	require.NoError(t, err)
	require.Equal(t, account.ID, validatedAccount.ID)
	require.Equal(t, account.Username, validatedAccount.Username)
	require.Equal(t, account.Email, validatedAccount.Email)
}

func TestAccountExists(t *testing.T) {
	repo := mock.New()

	id := uuid.MustParse("990014fe-b4d2-49f0-afa7-3118799ec3d4")
	guild := game.NewGuild("test")
	hero := game.NewHero("Test Hero")
	actor := model.AccountActor{
		ID:       id,
		Username: "test",
		Guild:    guild,
	}

	{
		err := repo.PersistAccountActor(context.Background(), actor)
		require.NoError(t, err)
	}

	{
		actor.Guild.Gold.Add(1000)
		err := repo.PersistAccountActor(context.Background(), actor)
		require.NoError(t, err)
	}

	{
		actor.Guild.AddHero(&hero)
		err := repo.PersistAccountActor(context.Background(), actor)
		require.NoError(t, err)
	}

	{
		for range 30 {
			actor.Guild.ProcessTick(context.Background())
		}

		err := repo.PersistAccountActor(context.Background(), actor)
		require.NoError(t, err)
	}

	{
		for len(hero.Inventory.Resources()) == 0 {
			guild.ProcessTick(context.Background())
		}

		err := repo.PersistAccountActor(context.Background(), actor)
		require.NoError(t, err)
	}

	restoredActor, err := repo.RestoreAccountActor(context.Background(), id)
	require.NoError(t, err)

	require.Equal(t, float64(4000), restoredActor.Guild.Gold.Load())
}
