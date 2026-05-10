package memory

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/alfreddobradi/actors/pkg/database"
	"github.com/alfreddobradi/actors/pkg/telemetry"
)

type Store struct {
	data map[string]string
	mu   sync.RWMutex
}

func NewStore() (*Store, error) {
	s := &Store{
		data: make(map[string]string),
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

func (s *Store) Persist(ctx context.Context, key string, value database.Snapshot) error {
	span := telemetry.SpanFromContext(ctx)
	span.GetLogger().Info("Persisting snapshot in database", "key", key, "timestamp", value.Timestamp)

	s.mu.Lock()
	defer s.mu.Unlock()

	s.data[key] = value.String()
	return nil
}

func (s *Store) Restore(ctx context.Context, key string) (database.Snapshot, error) {
	span := telemetry.SpanFromContext(ctx)
	span.GetLogger().Info("Restoring snapshot from database", "key", key)

	s.mu.RLock()
	defer s.mu.RUnlock()

	val, ok := s.data[key]
	if !ok {
		return database.Snapshot{}, fmt.Errorf("snapshot not found for key: %s", key)
	}

	var snapshot database.Snapshot
	aux := map[string]any{}
	err := json.Unmarshal([]byte(val), &aux)
	if err != nil {
		return database.Snapshot{}, fmt.Errorf("failed to unmarshal snapshot: %w", err)
	}

	if timestamp, ok := aux["timestamp"].(float64); ok {
		snapshot.Timestamp = int64(timestamp)
	}
	if data, ok := aux["data"].(string); ok {
		decoded, err := base64.URLEncoding.DecodeString(data)
		if err != nil {
			return database.Snapshot{}, fmt.Errorf("failed to decode snapshot data: %w", err)
		}
		snapshot.Data = decoded
	}

	return snapshot, nil
}

func init() {
	var _ database.DB = (*Store)(nil)
}
