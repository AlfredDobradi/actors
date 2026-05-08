package system

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync/atomic"

	"github.com/alfreddobradi/actors/examples/game/config"
	"github.com/alfreddobradi/actors/examples/game/database"
	"github.com/google/uuid"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type ContextKey string

const (
	ContextKeyError         ContextKey = "error"
	ContextKeySender        ContextKey = "sender"
	ContextKeySenderFn      ContextKey = "sender_fn"
	ContextKeySpanID        ContextKey = "span_id"
	ContextKeyFactoryParams ContextKey = "factory_params"
)

type ActorState uint8

const (
	ActorStateNotFound ActorState = iota
	ActorStateLocal
	ActorStateRemote
)

type ActorFactory func(ctx context.Context) Actor
type SenderFunc func(ctx context.Context, request bool, sender uuid.UUID, recipient Recipient, payload any) (any, error)
type HandlerOpt func(*ActorHandler, *System)

type MessageConsumer interface {
	Publish(ctx context.Context, sender uuid.UUID, recipient Recipient, payload any) error
	Request(ctx context.Context, sender uuid.UUID, recipient Recipient, payload any) error
}

type Actor interface {
	GetID() uuid.UUID
	GetKind() string

	Start(context.Context)
	Stop(context.Context) error
	HandleMessage(context.Context, *Message) HandleError

	Persist(context.Context, database.DB) error
	Restore(context.Context, database.DB) error
}

type ActorHandler struct {
	actor Actor

	inbox chan *Message
	stop  chan struct{}
	done  chan struct{}

	poisoned atomic.Bool

	preStartHooks   *HookCollection
	postStartHooks  *HookCollection
	poisonedHooks   *HookCollection
	terminatedHooks *HookCollection
	crashHooks      *HookCollection
}

func NewActorHandler(sys *System, actor Actor, opts ...HandlerOpt) *ActorHandler {
	handler := &ActorHandler{
		actor: actor,
		inbox: make(chan *Message, 100), // Buffered channel for messages

		stop: make(chan struct{}),
		done: make(chan struct{}),

		preStartHooks:   NewHookCollection(HookPreStart),
		postStartHooks:  NewHookCollection(HookPostStart),
		poisonedHooks:   NewHookCollection(HookPoisoned),
		terminatedHooks: NewHookCollection(HookTerminated),
		crashHooks:      NewHookCollection(HookCrash),
	}

	for _, opt := range opts {
		opt(handler, sys)
	}

	return handler
}

func (h *ActorHandler) GetActor() Actor {
	return h.actor
}

func (h *ActorHandler) GetID() uuid.UUID {
	return h.actor.GetID()
}

func (h *ActorHandler) GetKind() string {
	return h.actor.GetKind()
}

func (h *ActorHandler) GetAddress() string {
	return fmt.Sprintf("actor:%s", h.GetID())
}

func (h *ActorHandler) Stop() {
	close(h.stop)
}

func (h *ActorHandler) Start(ctx context.Context) {
	var handleError HandleError

	defer close(h.done)
	defer func() {
		// Run terminated hooks
		if h.terminatedHooks != nil {
			h.terminatedHooks.run(ctx, h.actor)
		}
	}()
	defer func() {
		// If the actor is terminating but wasn't poisoned it's an unexpected termination
		// TODO Add recovery logic (log, spawn new actor of the same kind, etc.)
		if !h.poisoned.Load() && h.crashHooks != nil {
			ctx = context.WithValue(ctx, ContextKeyError, handleError)
			h.crashHooks.run(ctx, h.actor)
			return
		}
	}()

	go h.GetActor().Start(ctx)

	for {
		select {
		case msg := <-h.inbox:
			if h.poisoned.Load() {
				slog.Warn("Actor is poisoned, ignoring message", "actor_id", h.actor.GetID())
				continue
			}

			// Handle incoming message
			slog.Debug("Actor received message", "actor_id", h.actor.GetID(), "message_id", msg.GetID(), "payload", fmt.Sprintf("%v", msg.GetBody()))
			handleError = h.actor.HandleMessage(ctx, msg)
			if handleError != nil && !handleError.IsRecoverable() {
				return
			}
		case <-h.stop:
			h.poisoned.Store(true)

			// Run poisoned hooks
			if h.poisonedHooks != nil {
				h.poisonedHooks.run(ctx, h.actor)
			}

			close(h.inbox)
			if err := h.GetActor().Stop(ctx); err != nil {
				slog.Warn("Error stopping actor", "actor_id", h.actor.GetID(), "error", err)
			}

			for m := range h.inbox {
				slog.Warn("Actor is poisoned, ignoring message", "actor_id", h.actor.GetID(), "message", fmt.Sprintf("%v", m.GetBody()))
			}

			// Handle actor termination
			return
		}
	}
}

func (h *ActorHandler) WaitForTermination() {
	<-h.done
}

func (h *ActorHandler) SendMessage(ctx context.Context, msg *Message) {
	if h.poisoned.Load() {
		slog.Warn("Cannot send message to poisoned actor", "actor_id", h.actor.GetID())
		return
	}

	h.inbox <- msg
}

type System struct {
	id   uuid.UUID
	name string

	bus      *Bus
	registry *Registry
	store    database.DB

	//nolint:unused
	transport any // TODO Add transport layer for external communication
}

func MustNewSystem(registry *Registry, store database.DB) *System {
	sys, err := NewSystem(registry, store)
	if err != nil {
		slog.Error("Failed to create system", "error", err)
		os.Exit(1)
	}
	return sys
}

func NewSystem(registry *Registry, store database.DB) (*System, error) {
	if store == nil {
		return nil, fmt.Errorf("database store is required")
	}

	if err := store.StartSession(context.Background()); err != nil {
		slog.Error("Failed to start session", "error", err)
		return nil, err
	}

	if registry == nil {
		registry = NewRegistry()
	}

	bus := NewBus()

	nodeName := config.GetConfig().NodeName
	id := uuid.NewSHA1(uuid.NameSpaceOID, []byte(nodeName))

	s := &System{
		name:     nodeName,
		id:       id,
		bus:      bus,
		registry: registry,
		store:    store,
	}

	bus.SetRouteFunction(func(actorID uuid.UUID, msg *Message) error {
		slog.Debug("Routing message", "actorID", actorID, "messageID", msg.GetID(), "payload", fmt.Sprintf("%v", msg.GetBody()))
		handler, exists := registry.actors[actorID]
		if !exists {
			slog.Warn("No actor found for subscription", "actorID", actorID)
			return nil
		}

		handler.SendMessage(context.Background(), msg)
		return nil
	})

	if err := s.advertise(context.Background(), config.GetConfig().Addr); err != nil {
		slog.Error("Failed to advertise system", "error", err)
		return nil, err
	}

	if err := s.startKeepalive(context.Background()); err != nil {
		slog.Error("Failed to start keepalive", "error", err)
		return nil, err
	}

	return s, nil
}

func (s *System) advertise(ctx context.Context, address string) error {
	if s.store != nil {
		if err := s.store.Set(database.WithLease(ctx, true), fmt.Sprintf("system:%s:hostname", s.id), database.ToStringerable(s.name)); err != nil {
			return err
		}
		if err := s.store.Set(database.WithLease(ctx, true), fmt.Sprintf("system:%s:address", s.id), database.ToStringerable(address)); err != nil {
			return err
		}
	}
	return nil
}

func (s *System) startKeepalive(ctx context.Context) error {
	if s.store != nil {
		return s.store.KeepAlive(ctx, func(data any) {
			switch v := data.(type) {
			case clientv3.LeaseKeepAliveResponse:
				slog.Debug("Received keepalive callback", "lease_id", v.ID, "data", fmt.Sprintf("%v", data))
			default:
				slog.Debug("Received unknown keepalive response", "data", fmt.Sprintf("%v", data))
			}
		})
	}
	return nil
}

func (s *System) Shutdown(ctx context.Context) error {
	if s.store != nil {
		if err := s.store.EndSession(ctx); err != nil {
			slog.Error("Failed to end database session", "error", err)
			return err
		}
	}
	return nil
}

func (s *System) GetName() string {
	return s.name
}

func (s *System) GetSystemID() uuid.UUID {
	return s.id
}

type Publisher interface {
	Subscribe(pattern string, actorID uuid.UUID) error
}

func (s *System) Subscribe(pattern string, actorID uuid.UUID) error {
	return s.bus.Subscribe(pattern, actorID)
}

func (s *System) Route(ctx context.Context, msg *Message) error {
	return s.bus.Route(ctx, msg)
}

func (s *System) Send(ctx context.Context, msg *Message) {
	s.bus.Inbox() <- msg
}

func (s *System) Request(ctx context.Context, sender uuid.UUID, recipient Recipient, payload any) (any, error) {
	return s.bus.Request(ctx, sender, recipient, payload)
}

func (s *System) Publish(ctx context.Context, sender uuid.UUID, recipient Recipient, payload any) error {
	return s.bus.Publish(ctx, sender, recipient, payload)
}

func (s *System) registerActor(ctx context.Context, handler *ActorHandler) error {
	s.registry.actors[handler.actor.GetID()] = handler
	if s.store != nil {
		if err := s.store.Set(database.WithLease(ctx, true), fmt.Sprintf("actor:%s:hostname", handler.actor.GetID()), database.ToStringerable(s.name)); err != nil {
			return err
		}
	}
	return nil
}

// TODO Unexpose this method once we have better communication options.
func (s *System) Spawn(ctx context.Context, kind string, opts ...HandlerOpt) (*ActorHandler, error) {
	factory, exists := s.registry.factories[kind]
	if !exists {
		return nil, fmt.Errorf("no factory registered for kind: %s", kind)
	}

	senderFn := SenderFunc(func(ctx context.Context, request bool, sender uuid.UUID, recipient Recipient, payload any) (any, error) {
		if request {
			return s.bus.Request(ctx, sender, recipient, payload)
		}

		err := s.bus.Publish(ctx, sender, recipient, payload)
		return nil, err
	})
	ctx = context.WithValue(ctx, ContextKeySenderFn, senderFn)

	actor := factory.Fn(ctx)

	if s.registry.actors[actor.GetID()] != nil {
		slog.Debug("Actor already exists with ID, returning existing handler", "actorID", actor.GetID())
		return s.registry.actors[actor.GetID()], nil
	}

	opts = append(opts, WithSubscription(fmt.Sprintf("actor:%s", actor.GetID())))
	opts = append(opts, WithSubscription(fmt.Sprintf("kind:%s", actor.GetKind())))

	handler := NewActorHandler(s, actor, opts...)

	for _, hookOpt := range factory.Hooks {
		hookOpt(handler, s)
	}

	if handler.preStartHooks != nil {
		handler.preStartHooks.run(ctx, actor)
	}

	go handler.Start(ctx)

	if handler.postStartHooks != nil {
		handler.postStartHooks.run(ctx, actor)
	}

	if err := s.registerActor(ctx, handler); err != nil {
		return nil, err
	}

	return handler, nil
}

func (s *System) AttemptRestoreActor(ctx context.Context, kind string, id uuid.UUID) (*ActorHandler, error) {
	factory, exists := s.registry.factories[kind]
	if !exists {
		return nil, fmt.Errorf("no factory registered for kind: %s", kind)
	}

	// Create a temporary actor instance to call Restore on
	actor := factory.Fn(ctx)

	if err := actor.Restore(ctx, s.store); err != nil {
		return nil, fmt.Errorf("failed to restore actor of kind %s with ID %s: %w", kind, id, err)
	}

	if s.registry.actors[actor.GetID()] != nil {
		slog.Debug("Actor already exists with ID, returning existing handler", "actorID", actor.GetID())
		return s.registry.actors[actor.GetID()], nil
	}

	opts := []HandlerOpt{
		WithSubscription(fmt.Sprintf("actor:%s", actor.GetID())),
		WithSubscription(fmt.Sprintf("kind:%s", actor.GetKind())),
	}

	handler := NewActorHandler(s, actor, opts...)

	if handler.preStartHooks != nil {
		handler.preStartHooks.run(ctx, actor)
	}

	go handler.Start(ctx)

	if handler.postStartHooks != nil {
		handler.postStartHooks.run(ctx, actor)
	}

	if err := s.registerActor(ctx, handler); err != nil {
		return nil, err
	}

	return handler, nil
}

func (s *System) SpawnWithParams(ctx context.Context, kind string, params any, opts ...HandlerOpt) (*ActorHandler, error) {
	ctx = context.WithValue(ctx, ContextKeyFactoryParams, params)
	return s.Spawn(ctx, kind, opts...)
}

func (s *System) IsActorSpawned(ctx context.Context, actorID uuid.UUID) ActorState {
	// lookup actor locally
	if _, exists := s.registry.actors[actorID]; exists {
		return ActorStateLocal
	}

	// If we have no connection to a store there's no way of looking up remote actors
	if s.store == nil {
		return ActorStateNotFound
	}

	// lookup actor in the store
	key := fmt.Sprintf("actor:%s:hostname", actorID)
	if _, ok := s.store.Get(ctx, key); ok {
		return ActorStateRemote
	}

	return ActorStateNotFound
}
