package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"testing"

	"github.com/alfreddobradi/actors/examples/game/actor"
	"github.com/alfreddobradi/actors/examples/game/api"
	"github.com/alfreddobradi/actors/examples/game/model"
	"github.com/alfreddobradi/actors/pkg/config"
	"github.com/alfreddobradi/actors/pkg/database/memory"
	"github.com/alfreddobradi/actors/pkg/system"
	"github.com/stretchr/testify/require"
)

// eventually this file will only contain a handful of end to end tests that covers the main happy paths like the flow from creating an account to retrieving character data

const (
	HOST     = "http://localhost:8080"
	USERNAME = "testuser"
	PASSWORD = "password123"
	EMAIL    = "test@email.com"
)

type response struct {
	StatusCode int
	Body       bytes.Buffer
}

func url(address, path string) string {
	return fmt.Sprintf("http://%s%s", address, path)
}

var usedPorts = make(map[int32]struct{})

func startSuite() (string, chan struct{}) {
	registry := system.NewRegistry()
	actor.InitFactories(registry)
	db := memory.NewStore()
	sys := system.MustNewSystem(registry, db)

	var port int32
	for {
		port = rand.Int31n(300) + 8100
		if _, exists := usedPorts[port]; !exists {
			usedPorts[port] = struct{}{}
			break
		}
	}

	address := fmt.Sprintf("localhost:%d", port)

	config.GetConfig().Addr = address
	s := api.NewServer(sys, db)

	stop := make(chan struct{})
	ctx := context.Background()

	go func() {
		s.Start()
		<-stop
		s.Shutdown(ctx)
		sys.Shutdown(ctx)
		db.Close(ctx)
	}()

	return address, stop
}

func createAccount(t *testing.T, address string) (response, error) {
	req, err := http.NewRequest(
		http.MethodPost,
		url(address, "/auth/account"),
		bytes.NewBufferString(`{"username":"`+USERNAME+`","email":"`+EMAIL+`","password":"`+PASSWORD+`"}`),
	)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	var buf bytes.Buffer
	_, err = buf.ReadFrom(resp.Body)
	require.NoError(t, err)

	return response{
		StatusCode: resp.StatusCode,
		Body:       buf,
	}, err
}

func createSession(t *testing.T, address string) (response, error) {
	req, err := http.NewRequest(
		http.MethodPost,
		url(address, "/auth/session"),
		bytes.NewBufferString(`{"username":"`+USERNAME+`","password":"`+PASSWORD+`"}`),
	)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	var buf bytes.Buffer
	_, err = buf.ReadFrom(resp.Body)
	require.NoError(t, err)

	return response{
		StatusCode: resp.StatusCode,
		Body:       buf,
	}, err
}

func createTavern(t *testing.T, address, token string) (response, error) {
	req, err := http.NewRequest(
		http.MethodPost,
		url(address, "/account/tavern"),
		nil,
	)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	var buf bytes.Buffer
	_, err = buf.ReadFrom(resp.Body)
	require.NoError(t, err)

	return response{
		StatusCode: resp.StatusCode,
		Body:       buf,
	}, err
}

func hireCharacter(t *testing.T, address, token string) (response, error) {
	req, err := http.NewRequest(
		http.MethodPost,
		url(address, "/account/tavern/characters/hire"),
		nil,
	)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	var buf bytes.Buffer
	_, err = buf.ReadFrom(resp.Body)
	require.NoError(t, err)

	return response{
		StatusCode: resp.StatusCode,
		Body:       buf,
	}, err
}

func getCharacter(t *testing.T, address, token, id string) (response, error) {
	req, err := http.NewRequest(
		http.MethodGet,
		url(address, fmt.Sprintf("/account/tavern/characters/%s", id)),
		nil,
	)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	var buf bytes.Buffer
	_, err = buf.ReadFrom(resp.Body)
	require.NoError(t, err)

	return response{
		StatusCode: resp.StatusCode,
		Body:       buf,
	}, err
}

func getCharacters(t *testing.T, address, token string) (response, error) {
	req, err := http.NewRequest(
		http.MethodGet,
		url(address, "/account/tavern/characters"),
		nil,
	)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	var buf bytes.Buffer
	_, err = buf.ReadFrom(resp.Body)
	require.NoError(t, err)

	return response{
		StatusCode: resp.StatusCode,
		Body:       buf,
	}, err
}

func TestApiIndex(t *testing.T) {
	address, stop := startSuite()
	defer close(stop)

	req, err := http.NewRequest(http.MethodGet, url(address, "/"), nil)
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	buf := bytes.NewBufferString("")
	_, err = buf.ReadFrom(resp.Body)
	require.NoError(t, err)
	require.Equal(t, "Welcome to the game API", buf.String())
}

func TestCreateAccount(t *testing.T) {
	address, stop := startSuite()
	defer close(stop)

	resp, err := createAccount(t, address)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	require.Contains(t, resp.Body.String(), `"username":"`+USERNAME+`"`)
}

func TestCreateSession(t *testing.T) {
	address, stop := startSuite()
	defer close(stop)

	_, err := createAccount(t, address)
	require.NoError(t, err)

	resp, err := createSession(t, address)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	require.Contains(t, resp.Body.String(), `"token":`)
}

func TestCreateTavern(t *testing.T) {
	address, stop := startSuite()
	defer close(stop)

	_, err := createAccount(t, address)
	require.NoError(t, err)

	sessionResp, err := createSession(t, address)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, sessionResp.StatusCode)

	var sessionData struct {
		Token string `json:"token"`
	}
	err = json.Unmarshal(sessionResp.Body.Bytes(), &sessionData)
	require.NoError(t, err)

	resp, err := createTavern(t, address, sessionData.Token)
	require.NoError(t, err)

	require.Equal(t, http.StatusOK, resp.StatusCode)

	require.Contains(t, resp.Body.String(), "Tavern creation requested successfully")
}

func TestGetCharacter(t *testing.T) {
	address, stop := startSuite()
	defer close(stop)

	acc, err := createAccount(t, address)
	require.NoError(t, err)

	resp, err := createSession(t, address)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var sessionResp struct {
		Token string `json:"token"`
	}
	err = json.Unmarshal(resp.Body.Bytes(), &sessionResp)
	require.NoError(t, err)

	var accResp struct {
		ID string `json:"id"`
	}
	err = json.Unmarshal(acc.Body.Bytes(), &accResp)
	require.NoError(t, err)

	_, err = createTavern(t, address, sessionResp.Token)
	require.NoError(t, err)

	resp, err = hireCharacter(t, address, sessionResp.Token)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var hireResp struct {
		OK            bool   `json:"ok"`
		CharacterID   string `json:"character_id"`
		CharacterName string `json:"character_name"`
		Error         string `json:"error,omitempty"`
	}
	err = json.Unmarshal(resp.Body.Bytes(), &hireResp)
	require.NoError(t, err)
	require.True(t, hireResp.OK)
	require.NotEmpty(t, hireResp.CharacterID)
	require.NotEmpty(t, hireResp.CharacterName)

	charRespRaw, err := getCharacter(t, address, sessionResp.Token, hireResp.CharacterID)
	require.NoError(t, err)

	var charResp struct {
		Status  string                 `json:"status"`
		Details model.CharacterDetails `json:"details,omitempty"`
		Error   string                 `json:"error,omitempty"`
	}
	err = json.Unmarshal(charRespRaw.Body.Bytes(), &charResp)
	require.NoError(t, err)
	require.Equal(t, "OK", charResp.Status)
	require.Equal(t, hireResp.CharacterID, charResp.Details.ID.String())
	require.Equal(t, hireResp.CharacterName, charResp.Details.Name)
	require.Equal(t, 1, charResp.Details.Level)
	require.Equal(t, 0, charResp.Details.Experience)

	multiResp, err := getCharacters(t, address, sessionResp.Token)
	require.NoError(t, err)

	var multiCharResp struct {
		Status  string                   `json:"status"`
		Details []model.CharacterDetails `json:"details,omitempty"`
		Error   string                   `json:"error,omitempty"`
	}
	err = json.Unmarshal(multiResp.Body.Bytes(), &multiCharResp)
	require.NoError(t, err)
	require.Equal(t, "OK", multiCharResp.Status)
	require.Len(t, multiCharResp.Details, 1)
	require.Equal(t, hireResp.CharacterID, multiCharResp.Details[0].ID.String())
	require.Equal(t, hireResp.CharacterName, multiCharResp.Details[0].Name)
	require.Equal(t, 1, multiCharResp.Details[0].Level)
	require.Equal(t, 0, multiCharResp.Details[0].Experience)
}
