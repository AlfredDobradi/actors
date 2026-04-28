package system

import (
	"context"
	"fmt"
	"log/slog"
	"sync/atomic"

	"github.com/google/uuid"
)

/*
* Registry of actors
* Bus for messaging between actors
* Ability to spawn actor ad-hoc for execution
* Lifecycle events
 */

type ContextKey string

const (
	ContextKeyError ContextKey = "error"
)

type ActorFactory func(ctx context.Context) Actor

type HandlerOpt func(*ActorHandler, *System)

type ActorOpt func(Actor)

type Message struct {
	ID      uuid.UUID
	Payload []byte
}

type Actor interface {
	GetID() uuid.UUID
	GetKind() string

	Start(context.Context)
	Stop(context.Context) error
	HandleMessage(context.Context, Message) HandleError
}

type ActorHandler struct {
	actor Actor

	inbox chan Message
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
		inbox: make(chan Message, 100), // Buffered channel for messages
		stop:  make(chan struct{}),
		done:  make(chan struct{}),

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
			slog.Debug("Actor received message", "actor_id", h.actor.GetID(), "message_id", msg.ID, "payload", string(msg.Payload))
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
				slog.Warn("Actor is poisoned, ignoring message", "actor_id", h.actor.GetID(), "message", string(m.Payload))
			}

			// Handle actor termination
			return
		}
	}
}

func (h *ActorHandler) WaitForTermination() {
	<-h.done
}

func (h *ActorHandler) SendMessage(ctx context.Context, msg Message) {
	if h.poisoned.Load() {
		slog.Warn("Cannot send message to poisoned actor", "actor_id", h.actor.GetID())
		return
	}

	h.inbox <- msg
}

type System struct {
	bus      *Bus
	registry *Registry

	//nolint:unused
	transport any // TODO Add transport layer for external communication
}

func NewSystem(bus *Bus, registry *Registry) *System {
	if bus == nil {
		bus = NewBus()
	}

	if registry == nil {
		registry = NewRegistry()
	}

	s := &System{
		bus:      bus,
		registry: registry,
	}

	bus.SetRouteFunction(func(actorID uuid.UUID, msg RouteableMessage) error {
		slog.Debug("Routing message", "actorID", actorID, "messageID", msg.GetID(), "payload", string(msg.GetBody()))
		handler, exists := registry.actors[actorID]
		if !exists {
			slog.Warn("No actor found for subscription", "actorID", actorID)
			return nil
		}

		handler.SendMessage(context.Background(), Message{
			ID:      msg.GetID(),
			Payload: msg.GetBody(),
		})
		return nil
	})

	return s
}

type Publisher interface {
	Subscribe(pattern string, actorID uuid.UUID) error
}

func (s *System) Subscribe(pattern string, actorID uuid.UUID) error {
	return s.bus.Subscribe(pattern, actorID)
}

func (s *System) Route(msg RouteableMessage) error {
	return s.bus.Route(msg)
}

// TODO Unexpose this method once we have better communication options.
func (s *System) Spawn(ctx context.Context, kind string, opts ...HandlerOpt) (*ActorHandler, error) {
	factory, exists := s.registry.factories[kind]
	if !exists {
		return nil, fmt.Errorf("no factory registered for kind: %s", kind)
	}

	actor := factory(ctx)

	handler := NewActorHandler(s, actor, opts...)

	if handler.preStartHooks != nil {
		handler.preStartHooks.run(ctx, actor)
	}

	go handler.Start(ctx)

	if handler.postStartHooks != nil {
		handler.postStartHooks.run(ctx, actor)
	}

	s.registry.actors[actor.GetID()] = handler

	return handler, nil
}
