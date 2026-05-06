package model

type ContextKey uint8

const (
	ContextKeyAccountData ContextKey = iota
	ContextKeySessionID
)
