package etcd

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/alfreddobradi/actors/pkg/database"
	"github.com/alfreddobradi/actors/pkg/telemetry"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type Store struct {
	*clientv3.Client

	sessionID int64
}

func New(endpoints []string) (*Store, error) {
	cfg := clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 2 * time.Second,
	}
	client, err := clientv3.New(cfg)
	if err != nil {
		return nil, err
	}
	return &Store{Client: client}, nil
}

func (s *Store) Set(ctx context.Context, key string, value fmt.Stringer) error {
	span := telemetry.SpanFromContext(ctx)
	span.GetLogger().Debug("Setting key in database", "key", key)

	fields := []any{"key", key}

	opts := []clientv3.OpOption{}
	if database.UseLease(ctx) {
		fields = append(fields, "lease_id", s.sessionID)
		opts = append(opts, clientv3.WithLease(clientv3.LeaseID(s.sessionID)))
	}

	slog.Debug("Setting key in database", fields...)

	_, err := s.Client.Put(context.Background(), key, value.String(), opts...)
	return err
}

func (s *Store) Get(ctx context.Context, key string) (string, bool) {
	span := telemetry.SpanFromContext(ctx)
	span.GetLogger().Info("Getting key from database", "key", key)

	opts := []clientv3.OpOption{}
	if database.UseLease(ctx) {
		opts = append(opts, clientv3.WithLease(clientv3.LeaseID(s.sessionID)))
	}

	resp, err := s.Client.Get(ctx, key, opts...)
	if err != nil || len(resp.Kvs) == 0 {
		return "", false
	}
	return string(resp.Kvs[0].Value), true
}

func (s *Store) Close(ctx context.Context) error {
	span := telemetry.SpanFromContext(ctx)
	span.GetLogger().Info("Closing database connection")

	return s.Client.Close()
}

func (s *Store) Delete(ctx context.Context, key string) error {
	span := telemetry.SpanFromContext(ctx)
	span.GetLogger().Info("Deleting key from database", "key", key)

	opts := []clientv3.OpOption{}
	if database.UseLease(ctx) {
		opts = append(opts, clientv3.WithLease(clientv3.LeaseID(s.sessionID)))
	}

	_, err := s.Client.Delete(ctx, key, opts...)
	return err
}

func (s *Store) Keys(ctx context.Context) []string {
	opts := []clientv3.OpOption{
		clientv3.WithPrefix(),
		clientv3.WithKeysOnly(),
	}
	if database.UseLease(ctx) {
		opts = append(opts, clientv3.WithLease(clientv3.LeaseID(s.sessionID)))
	}

	resp, err := s.Client.Get(ctx, "", opts...)
	if err != nil {
		return nil
	}
	keys := make([]string, len(resp.Kvs))
	for i, kv := range resp.Kvs {
		keys[i] = string(kv.Key)
	}
	return keys
}

func (s *Store) Persist(ctx context.Context) error {
	return nil
}

func (s *Store) Load(ctx context.Context) error {
	return nil
}

func init() {
	var _ database.DB = (*Store)(nil)
}

func (s *Store) StartSession(ctx context.Context) error {
	resp, err := s.Client.Grant(ctx, 10) // 10 second TTL
	if err != nil {
		return err
	}
	s.sessionID = int64(resp.ID)

	slog.Info("Starting database session", "session_id", s.sessionID)

	return nil
}

func (s *Store) KeepAlive(ctx context.Context, callback func(any)) error {
	defer slog.Info("Stopping database keepalive", "session_id", s.sessionID)
	ch, err := s.Client.KeepAlive(ctx, clientv3.LeaseID(s.sessionID))
	if err != nil {
		return err
	}
	for resp := range ch {
		if resp == nil {
			slog.Warn("Keepalive channel closed", "session_id", s.sessionID)
			return nil
		}
		callback(resp)
	}
	return nil
}

func (s *Store) EndSession(ctx context.Context) error {
	_, err := s.Client.Revoke(ctx, clientv3.LeaseID(s.sessionID))

	slog.Info("Ending database session", "session_id", s.sessionID)

	return err
}
