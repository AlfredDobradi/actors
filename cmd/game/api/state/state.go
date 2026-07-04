package state

import (
	"github.com/alfreddobradi/actors/pkg/database"
	"github.com/alfreddobradi/actors/pkg/database/postgres"
	"github.com/alfreddobradi/actors/pkg/system"
)

type Context struct {
	KV     database.KeyValue
	DB     *postgres.Connection
	System *system.System
}

func New(kv database.KeyValue, db *postgres.Connection, sys *system.System) Context {
	return Context{
		KV:     kv,
		DB:     db,
		System: sys,
	}
}
