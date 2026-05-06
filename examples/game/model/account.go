package model

import (
	"bytes"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type CreateAccountRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type CreateAccountResponse struct {
	ID       uuid.UUID `json:"id"`
	Username string    `json:"username"`
	Email    string    `json:"email"`
}

type Account struct {
	ID        uuid.UUID `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Password  string    `json:"password"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Active    bool      `json:"active"`
}

func (a Account) String() string {
	buf := bytes.NewBufferString("")
	encoder := json.NewEncoder(buf)
	encoder.Encode(a)
	return buf.String()
}

type CreateSessionRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type CreateSessionResponse struct {
	ID    uuid.UUID `json:"id"`
	Token string    `json:"token"`
}

type Session struct {
	ID        uuid.UUID `json:"id"`
	AccountID uuid.UUID `json:"account_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Active    bool      `json:"active"`
}

func (s Session) String() string {
	buf := bytes.NewBufferString("")
	encoder := json.NewEncoder(buf)
	encoder.Encode(s)
	return buf.String()
}
