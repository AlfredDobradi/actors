package postgres

import (
	"github.com/alfreddobradi/actors/pkg/database"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type Connection struct {
	*sqlx.DB
}

func New(dsn string) (*Connection, error) {
	c, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		return nil, err
	}

	return &Connection{
		DB: c,
	}, nil
}

func init() {
	var _ database.Store = (*Connection)(nil)
}
