package state

import (
	"github.com/alfreddobradi/actors/pkg/database"
	"github.com/alfreddobradi/actors/pkg/system"
)

type Context struct {
	DB     database.DB
	System *system.System
}

func New(db database.DB, sys *system.System) Context {
	return Context{
		DB:     db,
		System: sys,
	}
}
