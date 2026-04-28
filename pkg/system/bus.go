package system

import (
	"log/slog"
	"sync"

	"github.com/google/uuid"
)

type RouteableMessage interface {
	GetID() uuid.UUID
	GetTopic() string
	GetBody() []byte
}

type Bus struct {
	inbox     chan any
	subscribe chan Subscription

	subscriptions []Subscription
	routeFn       func(actorID uuid.UUID, msg RouteableMessage) error
}

func NewBus() *Bus {
	return &Bus{
		inbox:         make(chan any, 100),
		subscribe:     make(chan Subscription, 100),
		subscriptions: make([]Subscription, 0),
		routeFn:       nil,
	}
}

func (b *Bus) Route(msg RouteableMessage) error {
	for _, sub := range b.subscriptions {
		matches := sub.pattern.MatchString(msg.GetTopic())
		slog.Debug("Routing message", "messageID", msg.GetID(), "topic", msg.GetTopic(), "subscriptionPattern", sub.pattern.String(), "matches", matches)
		if matches {
			if b.routeFn != nil {
				err := b.routeFn(sub.actorID, msg)
				if err != nil {
					slog.Error("Failed to route message", "messageID", msg.GetID(), "error", err)
					return err
				}
			} else {
				slog.Warn("No routing function defined, message will not be delivered", "messageID", msg.GetID())
			}
		}
	}

	return nil
}

func (b *Bus) SetRouteFunction(routeFn func(actorID uuid.UUID, msg RouteableMessage) error) {
	b.routeFn = routeFn
}

func (b *Bus) Subscribe(pattern string, actorID uuid.UUID) error {
	sub, err := NewSubscription(pattern, actorID)
	if err != nil {
		return err
	}

	b.subscriptions = append(b.subscriptions, *sub)
	return nil
}

type MessageQueue struct {
	mx *sync.Mutex

	messages []any
}

func NewMessageQueue() *MessageQueue {
	return &MessageQueue{
		mx:       &sync.Mutex{},
		messages: make([]any, 0),
	}
}

func (q *MessageQueue) Len() int {
	q.mx.Lock()
	defer q.mx.Unlock()
	return len(q.messages)
}

func (q *MessageQueue) IsEmpty() bool {
	return q.Len() == 0
}

func (q *MessageQueue) Push(msg any) {
	q.mx.Lock()
	defer q.mx.Unlock()
	q.messages = append(q.messages, msg)
}

func (q *MessageQueue) Pop() any {
	q.mx.Lock()
	defer q.mx.Unlock()
	if len(q.messages) == 0 {
		return nil
	}
	msg := q.messages[0]
	q.messages = q.messages[1:]
	return msg
}
