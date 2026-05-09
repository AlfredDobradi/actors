package database

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type contextKey uint8

const (
	useLeaseKey contextKey = iota
)

func WithLease(ctx context.Context, use bool) context.Context {
	return context.WithValue(ctx, useLeaseKey, use)
}

func UseLease(ctx context.Context) bool {
	v, ok := ctx.Value(useLeaseKey).(bool)
	if !ok {
		return false
	}
	return v
}

type Stringerable struct {
	value string
}

func (s Stringerable) String() string {
	return s.value
}

func ToStringerable(value string) Stringerable {
	return Stringerable{value: value}
}

type Snapshot struct {
	Timestamp int64
	Data      []byte
}

func SnapshotKey(kind string, id uuid.UUID) string {
	return fmt.Sprintf("snapshot:%s:%s", kind, id)
}

func NewSnapshot(data []byte) Snapshot {
	return Snapshot{
		Timestamp: time.Now().Unix(),
		Data:      data,
	}
}

func (s Snapshot) String() string {
	data := base64.URLEncoding.EncodeToString(s.Data)
	snapshot := map[string]any{
		"timestamp": s.Timestamp,
		"data":      data,
	}

	encoded, err := json.Marshal(snapshot)
	if err != nil {
		return ""
	}
	return string(encoded)
}

type DB interface {
	Set(ctx context.Context, key string, value fmt.Stringer) error
	Get(ctx context.Context, key string) (string, bool)
	Delete(ctx context.Context, key string) error
	Keys(ctx context.Context) []string
	Close(ctx context.Context) error
	StartSession(ctx context.Context) error
	KeepAlive(ctx context.Context, callback func(any)) error
	EndSession(ctx context.Context) error
	Persist(ctx context.Context, key string, value Snapshot) error
	Restore(ctx context.Context, key string) (Snapshot, error)
}
