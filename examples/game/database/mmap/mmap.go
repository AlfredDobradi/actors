package mmap

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"sync"
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
		if err := s.Load(); err != nil && !os.IsNotExist(err) {
			return nil, err
		}
	}

	return s, nil
}

func (s *Store) Set(key string, value fmt.Stringer) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data[key] = value.String()
	return nil
}

func (s *Store) Get(key string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	val, ok := s.data[key]
	return val, ok
}

func (s *Store) Delete(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.data, key)
	return nil
}

func (s *Store) Clear() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data = make(map[string]string)
	return nil
}

func (s *Store) Keys() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	keys := make([]string, 0, len(s.data))
	for key := range s.data {
		keys = append(keys, key)
	}
	return keys
}

func (s *Store) Persist() error {
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

	return os.WriteFile(s.path, buf.Bytes(), 0644)
}

func (s *Store) Load() error {
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

func (s *Store) Size() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.data)
}
