package system

import (
	"context"
	"fmt"
	"sync"
)

type FactoryRegistry struct {
	mx *sync.Mutex

	registry map[string]ActorFactory
}

var once sync.Once

var registry *FactoryRegistry

func GetRegistry() *FactoryRegistry {
	once.Do(func() {
		registry = &FactoryRegistry{
			mx:       &sync.Mutex{},
			registry: make(map[string]ActorFactory),
		}
	})
	return registry
}

func (r *FactoryRegistry) Register(kind string, factory ActorFactory) {
	r.mx.Lock()
	defer r.mx.Unlock()
	r.registry[kind] = factory
}

func (r *FactoryRegistry) Deregister(kind string) {
	r.mx.Lock()
	defer r.mx.Unlock()
	delete(r.registry, kind)
}

func (r *FactoryRegistry) Clear() {
	r.mx.Lock()
	defer r.mx.Unlock()
	r.registry = make(map[string]ActorFactory)
}

func (r *FactoryRegistry) Spawn(kind string, opts ...HandlerOpt) (*ActorHandler, error) {
	r.mx.Lock()
	defer r.mx.Unlock()
	factory, exists := r.registry[kind]
	if !exists {
		return nil, fmt.Errorf("no factory registered for kind: %s", kind)
	}
	return Spawn(context.Background(), factory, opts...), nil
}

func WithPreStartHook(hook Hook) HandlerOpt {
	return func(h *ActorHandler) {
		if h.preStartHooks == nil {
			h.preStartHooks = NewHookCollection("preStart")
		}
		h.preStartHooks.Add(hook)
	}
}

func WithPostStartHook(hook Hook) HandlerOpt {
	return func(h *ActorHandler) {
		if h.postStartHooks == nil {
			h.postStartHooks = NewHookCollection("postStart")
		}
		h.postStartHooks.Add(hook)
	}
}

func WithPoisonedHook(hook Hook) HandlerOpt {
	return func(h *ActorHandler) {
		if h.poisonedHooks == nil {
			h.poisonedHooks = NewHookCollection("poisoned")
		}
		h.poisonedHooks.Add(hook)
	}
}

func WithTerminatedHook(hook Hook) HandlerOpt {
	return func(h *ActorHandler) {
		if h.terminatedHooks == nil {
			h.terminatedHooks = NewHookCollection("terminated")
		}
		h.terminatedHooks.Add(hook)
	}
}
