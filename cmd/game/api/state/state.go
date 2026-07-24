package state

import (
	"github.com/alfreddobradi/actors/pkg/database"
	"github.com/alfreddobradi/actors/pkg/system"
)

type Context struct {
	KV     database.KeyValue
	DB     database.Store
	System *system.System
}

func New(kv database.KeyValue, db database.Store, sys *system.System) Context {
	return Context{
		KV:     kv,
		DB:     db,
		System: sys,
	}
}
