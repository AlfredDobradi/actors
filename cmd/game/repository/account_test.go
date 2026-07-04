package repository

// func TestCreateAccount(t *testing.T) {
// 	db := memory.NewStore()

// 	req := model.CreateAccountRequest{
// 		Username: "testuser",
// 		Email:    "testuser@example.com",
// 		Password: "password123",
// 	}

// 	resp, err := CreateAccount(context.Background(), db, req)
// 	require.NoError(t, err)
// 	require.Equal(t, req.Username, resp.Username)
// 	require.Equal(t, req.Email, resp.Email)

// 	// Verify account is stored in the database
// 	storedAccount, exists := db.Get(context.Background(), "account:"+resp.ID.String(), false)
// 	require.True(t, exists)
// 	require.NotNil(t, storedAccount)
// }

// func TestValidateCredentials(t *testing.T) {
// 	db := memory.NewStore()

// 	// Create a test account
// 	account := model.Account{
// 		ID:        uuid.New(),
// 		Username:  "testuser",
// 		Email:     "testuser@example.com",
// 		Password:  "password123",
// 		CreatedAt: time.Now(),
// 		UpdatedAt: time.Now(),
// 		Active:    true,
// 	}

// 	// Store the test account in the database
// 	err := db.Set(context.Background(), "account:"+account.ID.String(), account)
// 	require.NoError(t, err)

// 	// Validate credentials
// 	req := model.CreateSessionRequest{
// 		Username: "testuser",
// 		Password: "password123",
// 	}

// 	validatedAccount, err := ValidateCredentials(context.Background(), db, req)
// 	require.NoError(t, err)
// 	require.Equal(t, account.ID, validatedAccount.ID)
// 	require.Equal(t, account.Username, validatedAccount.Username)
// 	require.Equal(t, account.Email, validatedAccount.Email)
// }

// func TestAccountExists(t *testing.T) {
// 	db, err := postgres.New()
// 	require.NoError(t, err)

// 	id := uuid.MustParse("990014fe-b4d2-49f0-afa7-3118799ec3d4")
// 	guild := game.NewGuild("test")
// 	hero := game.NewHero("Test Hero")

// 	{
// 		tx, err := db.Beginx()
// 		require.NoError(t, err)

// 		err = persistGuildData(context.Background(), tx, id, guild)
// 		require.NoError(t, err)

// 		tx.Commit()
// 	}

// 	{
// 		tx, err := db.Beginx()
// 		require.NoError(t, err)

// 		time.Sleep(1 * time.Second)
// 		guild.Gold.Add(1000)
// 		err = persistGuildData(context.Background(), tx, id, guild)
// 		require.NoError(t, err)

// 		tx.Commit()
// 	}

// 	{
// 		tx, err := db.Beginx()
// 		require.NoError(t, err)

// 		time.Sleep(1 * time.Second)
// 		guild.AddCharacter(&hero)
// 		err = persistGuildData(context.Background(), tx, id, guild)
// 		require.NoError(t, err)

// 		tx.Commit()
// 	}

// 	{
// 		tx, err := db.Beginx()
// 		require.NoError(t, err)

// 		time.Sleep(1 * time.Second)

// 		for range 30 {
// 			guild.ProcessTick(context.Background())
// 		}

// 		err = persistGuildData(context.Background(), tx, id, guild)
// 		require.NoError(t, err)

// 		tx.Commit()
// 	}

// 	{
// 		tx, err := db.Beginx()
// 		require.NoError(t, err)

// 		time.Sleep(1 * time.Second)

// 		for {
// 			if len(hero.Inventory.Resources()) > 0 {
// 				break
// 			}

// 			guild.ProcessTick(context.Background())
// 		}

// 		err = persistGuildData(context.Background(), tx, id, guild)
// 		require.NoError(t, err)

// 		tx.Commit()
// 	}

// 	// spew.Dump(accounts)
// }
