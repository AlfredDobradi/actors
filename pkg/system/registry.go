package system

import (
	"fmt"
	"regexp"
	"sync"

	"github.com/google/uuid"
)

type Subscription struct {
	pattern regexp.Regexp
	actorID uuid.UUID
}

func NewSubscription(pattern string, actorID uuid.UUID) (*Subscription, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid subscription pattern: %w", err)
	}
	return &Subscription{
		pattern: *re,
		actorID: actorID,
	}, nil
}

type Registry struct {
	mx *sync.Mutex

	actors    map[uuid.UUID]*ActorHandler
	factories map[string]ActorFactory
}

func NewRegistry() *Registry {
	return &Registry{
		mx:        &sync.Mutex{},
		actors:    make(map[uuid.UUID]*ActorHandler),
		factories: make(map[string]ActorFactory),
	}
}

func (r *Registry) RegisterFactory(kind string, factory ActorFactory) {
	r.mx.Lock()
	defer r.mx.Unlock()
	r.factories[kind] = factory
}

func (r *Registry) DeregisterFactory(kind string) {
	r.mx.Lock()
	defer r.mx.Unlock()
	delete(r.factories, kind)
}

func (r *Registry) Clear() {
	r.mx.Lock()
	defer r.mx.Unlock()
	r.factories = make(map[string]ActorFactory)
}
