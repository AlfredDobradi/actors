package mmap

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/alfreddobradi/actors/pkg/database"
	"github.com/alfreddobradi/actors/pkg/telemetry"
)

type Store struct {
	data map[string]string
	mu   sync.RWMutex
	path string
}

func NewStore(path string) (*Store, error) {
	s := &Store{
		data: make(map[string]string),
		path: path,
	}

	if path != "" {
		if err := s.Load(context.Background()); err != nil && !os.IsNotExist(err) {
			return nil, err
		}
	}

	return s, nil
}

func (s *Store) Set(ctx context.Context, key string, value fmt.Stringer) error {
	span := telemetry.SpanFromContext(ctx)
	span.GetLogger().Debug("Setting key in database", "key", key)

	s.mu.Lock()
	defer s.mu.Unlock()

	s.data[key] = value.String()
	return nil
}

func (s *Store) Get(ctx context.Context, key string) (string, bool) {
	span := telemetry.SpanFromContext(ctx)
	span.GetLogger().Info("Getting key from database", "key", key)

	s.mu.RLock()
	defer s.mu.RUnlock()

	val, ok := s.data[key]
	return val, ok
}

func (s *Store) Delete(ctx context.Context, key string) error {
	span := telemetry.SpanFromContext(ctx)
	span.GetLogger().Info("Deleting key from database", "key", key)

	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.data, key)
	return nil
}

func (s *Store) Keys(ctx context.Context) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	keys := make([]string, 0, len(s.data))
	for key := range s.data {
		keys = append(keys, key)
	}
	return keys
}

func (s *Store) Persist(ctx context.Context) error {
	if s.path == "" {
		return nil
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	buf := bytes.NewBufferString("")
	encoder := json.NewEncoder(buf)
	if err := encoder.Encode(s.data); err != nil {
		return err
	}

	return os.WriteFile(s.path, buf.Bytes(), 0600)
}

func (s *Store) Load(ctx context.Context) error {
	if s.path == "" {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.path)
	if err != nil {
		return err
	}

	strs := make(map[string]string)
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(&strs); err != nil {
		return err
	}

	s.data = strs

	return nil
}

func (s *Store) Close(ctx context.Context) error {
	return nil
}

func (s *Store) StartSession(ctx context.Context) error {
	return nil
}

func (s *Store) KeepAlive(ctx context.Context, callback func(any)) error {
	return nil
}

func (s *Store) EndSession(ctx context.Context) error {
	return nil
}

func init() {
	var _ database.DB = (*Store)(nil)
}
