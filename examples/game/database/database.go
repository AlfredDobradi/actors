package database

import (
	"context"
	"fmt"
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

type DB interface {
	Set(ctx context.Context, key string, value fmt.Stringer) error
	Get(ctx context.Context, key string) (string, bool)
	Delete(ctx context.Context, key string) error
	Keys(ctx context.Context) []string
	Persist(ctx context.Context) error
	Load(ctx context.Context) error
	Close(ctx context.Context) error
	StartSession(ctx context.Context) error
	KeepAlive(ctx context.Context, callback func(any)) error
	EndSession(ctx context.Context) error
}
