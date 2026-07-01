package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

const (
	HOST     = "http://localhost:8080"
	USERNAME = "brvy"
	PASSWORD = "1234"
	EMAIL    = "test@password.dev"
)

type response struct {
	StatusCode int
	Body       bytes.Buffer
}

func url(path string) string {
	return fmt.Sprintf("%s%s", HOST, path)
}

func main() {
	_, err := createAccount()
	must(err)

	sessionResp, err := createSession()
	must(err)

	var sessionData struct {
		Token string `json:"token"`
	}
	err = json.Unmarshal(sessionResp.Body.Bytes(), &sessionData)
	must(err)

	fmt.Printf("Session token:\n\n%s\n\n", sessionData.Token)

	_, err = createTavern(sessionData.Token)
	must(err)

	_, err = hireCharacter(sessionData.Token)
	must(err)
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func createAccount() (response, error) {
	req, err := http.NewRequest(
		http.MethodPost,
		url("/auth/account"),
		bytes.NewBufferString(`{"username":"`+USERNAME+`","email":"`+EMAIL+`","password":"`+PASSWORD+`"}`),
	)
	must(err)

	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	must(err)
	defer resp.Body.Close()

	var buf bytes.Buffer
	_, err = buf.ReadFrom(resp.Body)
	must(err)

	return response{
		StatusCode: resp.StatusCode,
		Body:       buf,
	}, err
}

func createSession() (response, error) {
	req, err := http.NewRequest(
		http.MethodPost,
		url("/auth/session"),
		bytes.NewBufferString(`{"username":"`+USERNAME+`","password":"`+PASSWORD+`"}`),
	)
	must(err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	must(err)
	defer resp.Body.Close()

	var buf bytes.Buffer
	_, err = buf.ReadFrom(resp.Body)
	must(err)

	return response{
		StatusCode: resp.StatusCode,
		Body:       buf,
	}, err
}

func createTavern(token string) (response, error) {
	req, err := http.NewRequest(
		http.MethodPost,
		url("/account/tavern"),
		bytes.NewBufferString(`{"name":"Test Tavern"}`),
	)
	must(err)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	must(err)
	defer resp.Body.Close()

	var buf bytes.Buffer
	_, err = buf.ReadFrom(resp.Body)
	must(err)

	return response{
		StatusCode: resp.StatusCode,
		Body:       buf,
	}, err
}

func hireCharacter(token string) (response, error) {
	req, err := http.NewRequest(
		http.MethodPost,
		url("/account/tavern/characters/hire"),
		nil,
	)
	must(err)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	must(err)
	defer resp.Body.Close()

	var buf bytes.Buffer
	_, err = buf.ReadFrom(resp.Body)
	must(err)

	return response{
		StatusCode: resp.StatusCode,
		Body:       buf,
	}, err
}
