package postgres

import "github.com/alfreddobradi/actors/pkg/database/store/postgres"

type Repository struct {
	db *postgres.Connection
}

func New(db *postgres.Connection) *Repository {
	return &Repository{
		db: db,
	}
}
