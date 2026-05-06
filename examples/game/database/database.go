package database

import "fmt"

type DB interface {
	Set(key string, value fmt.Stringer) error
	Get(key string) (string, bool)
	Delete(key string) error
	Clear() error
	Keys() []string
	Persist() error
	Load() error
	Size() int
}
