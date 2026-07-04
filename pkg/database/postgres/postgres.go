package postgres

import (
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type Connection struct {
	*sqlx.DB
}

func New() (*Connection, error) {
	c, err := sqlx.Connect("postgres", "host=host.docker.internal user=postgres password=testing dbname=postgres sslmode=disable port=55432")
	if err != nil {
		return nil, err
	}

	return &Connection{
		DB: c,
	}, nil
}
