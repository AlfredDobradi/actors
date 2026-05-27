package model

import (
	"bytes"
	"encoding/json"
	"log/slog"
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
	if err := encoder.Encode(a); err != nil {
		slog.Error("Failed to encode account to JSON", "error", err, "account_id", a.ID)
		return ""
	}
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
	if err := encoder.Encode(s); err != nil {
		slog.Error("Failed to encode session to JSON", "error", err, "session_id", s.ID)
		return ""
	}
	return buf.String()
}

type SpawnAccountActorRequest struct {
	AccountID uuid.UUID `json:"account_id"`
}

type SpawnAccountActorResponse struct {
	AccountID uuid.UUID `json:"account_id"`
}

type AccountActorParams struct {
	ID   uuid.UUID
	Name string
}

func (p AccountActorParams) GetID() uuid.UUID {
	return p.ID
}
