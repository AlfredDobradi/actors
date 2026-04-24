package system

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

/*
* Registry of actors
* Bus for messaging between actors
* Ability to spawn actor ad-hoc for execution
* Lifecycle events
 */

type Hook func(context.Context, Actor) error

type HookCollection struct {
	name  string
	hooks []Hook
}

func (hc *HookCollection) Add(hook Hook) {
	hc.hooks = append(hc.hooks, hook)
}

func NewHookCollection(name string, hooks ...Hook) *HookCollection {
	return &HookCollection{
		name:  name,
		hooks: hooks,
	}
}

func (hc *HookCollection) run(_ context.Context, actor Actor) {
	subContext, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	wg := &sync.WaitGroup{}
	for _, h := range hc.hooks {
		wg.Add(1)
		go func(hook Hook) {
			defer wg.Done()
			if err := hook(subContext, actor); err != nil {
				slog.Warn("Error running hook", "hook", hc.name, "error", err)
			}
		}(h)
	}
	wg.Wait()

	if err := subContext.Err(); err != nil {
		slog.Warn("Hook execution context error", "hook", hc.name, "error", err)
		return
	}

	slog.Debug("Completed running hooks", "hook", hc.name, "count", len(hc.hooks))
}

type ActorFactory func(ctx context.Context) Actor

type HandlerOpt func(*ActorHandler)

type ActorOpt func(Actor)

type Message struct {
	ID      uuid.UUID
	Payload []byte
}

type Actor interface {
	GetID() uuid.UUID
	GetKind() string

	HandleMessage(context.Context, Message) error
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
}

func NewActorHandler(actor Actor, opts ...HandlerOpt) *ActorHandler {
	handler := &ActorHandler{
		actor: actor,
		inbox: make(chan Message, 100), // Buffered channel for messages
		stop:  make(chan struct{}),
		done:  make(chan struct{}),
	}

	for _, opt := range opts {
		opt(handler)
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
		if !h.poisoned.Load() {
			slog.Error("Unexpected actor termination", "actor_id", h.actor.GetID())
			return
		}
	}()

	for {
		select {
		case msg := <-h.inbox:
			if h.poisoned.Load() {
				slog.Warn("Actor is poisoned, ignoring message", "actor_id", h.actor.GetID())
				continue
			}

			// Handle incoming message
			if err := h.actor.HandleMessage(ctx, msg); err != nil {
				slog.Warn("Error handling message", "error", err)
			}
		case <-h.stop:
			h.poisoned.Store(true)

			// Run poisoned hooks
			if h.poisonedHooks != nil {
				h.poisonedHooks.run(ctx, h.actor)
			}

			close(h.inbox)

			for m := range h.inbox {
				slog.Warn("Actor is stopping, ignoring message", "actor_id", h.actor.GetID(), "message", m)
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

func Spawn(ctx context.Context, fn ActorFactory, opts ...HandlerOpt) *ActorHandler {
	actor := fn(ctx)

	handler := NewActorHandler(actor, opts...)

	if handler.preStartHooks != nil {
		handler.preStartHooks.run(ctx, actor)
	}

	go handler.Start(ctx)

	if handler.postStartHooks != nil {
		handler.postStartHooks.run(ctx, actor)
	}

	return handler
}
