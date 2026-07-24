package mock

import (
	"github.com/alfreddobradi/actors/cmd/game/model"
	"github.com/google/uuid"
)

type Repository struct {
	accounts      map[uuid.UUID]model.Account
	sessions      map[uuid.UUID]model.Session
	accountActors map[uuid.UUID]model.AccountActor
}

func New() *Repository {
	return &Repository{
		accounts:      make(map[uuid.UUID]model.Account),
		sessions:      make(map[uuid.UUID]model.Session),
		accountActors: make(map[uuid.UUID]model.AccountActor),
	}
}
