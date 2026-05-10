package model

import "github.com/google/uuid"

type ContextKey string

const (
	ContextKeyError         ContextKey = "error"
	ContextKeySender        ContextKey = "sender"
	ContextKeySenderFn      ContextKey = "sender_fn"
	ContextKeySpanID        ContextKey = "span_id"
	ContextKeyFactoryParams ContextKey = "factory_params"
	ContextKeyAccountID     ContextKey = "account_id"
)

type ActorState uint8

const (
	ActorStateNotFound ActorState = iota
	ActorStateLocal
	ActorStateRemote
)

type IDParam interface {
	GetID() uuid.UUID
}
