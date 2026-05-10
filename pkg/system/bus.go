package system

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/alfreddobradi/actors/pkg/model"
	"github.com/google/uuid"
)

type RecipientKind string

const (
	RecipientKindTopic RecipientKind = "topic"
	RecipientKindActor RecipientKind = "actor"
)

type Recipient struct {
	Kind    RecipientKind
	Subject string
}

func (r Recipient) String() string {
	return fmt.Sprintf("%s:%s", r.Kind, r.Subject)
}

// Message represents a message that can be sent between actors.
type Message struct {
	ID              uuid.UUID
	Sender          uuid.UUID
	Payload         any
	Recipient       Recipient
	ResponseTo      uuid.UUID
	ResponseChannel chan *Message
}

func (m *Message) GetID() uuid.UUID         { return m.ID }
func (m *Message) GetSender() uuid.UUID     { return m.Sender }
func (m *Message) GetBody() any             { return m.Payload }
func (m *Message) GetRecipient() Recipient  { return m.Recipient }
func (m *Message) GetResponseTo() uuid.UUID { return m.ResponseTo }
func (m *Message) IsRequest() bool          { return m.ResponseChannel != nil }
func (m *Message) Respond(sender uuid.UUID, payload any) error {
	if m.ResponseChannel == nil {
		slog.Debug("Message does not expect a response, ignoring respond call", "messageID", m.GetID())
		return nil
	}

	response := &Message{
		ID:              uuid.New(),
		Sender:          sender,
		Payload:         payload,
		Recipient:       Recipient{Kind: RecipientKindActor, Subject: m.GetSender().String()},
		ResponseTo:      m.GetID(),
		ResponseChannel: nil,
	}

	slog.Debug("Sending response message", "messageID", response.GetID(), "responseTo", response.GetResponseTo(), "payload", fmt.Sprintf("%v", payload))

	m.ResponseChannel <- response
	return nil
}

type SubscriptionGroup struct {
	id      uuid.UUID
	pattern *regexp.Regexp
	actors  map[uuid.UUID]struct{}
}

func NewSubscriptionGroup(pattern string) (*SubscriptionGroup, error) {
	// generate an id deterministically based on the pattern
	id := uuid.NewSHA1(uuid.NameSpaceOID, []byte(pattern))

	compiledPattern, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to compile subscription pattern: %w", err)
	}

	return &SubscriptionGroup{
		id:      id,
		pattern: compiledPattern,
		actors:  make(map[uuid.UUID]struct{}),
	}, nil
}

func (g *SubscriptionGroup) AddActor(actorID uuid.UUID) {
	g.actors[actorID] = struct{}{}
}

func (g *SubscriptionGroup) GetID() uuid.UUID {
	return g.id
}

func (g *SubscriptionGroup) GetPattern() *regexp.Regexp {
	return g.pattern
}

func (g *SubscriptionGroup) GetActors() map[uuid.UUID]struct{} {
	return g.actors
}

type Bus struct {
	inbox chan any
	stop  chan struct{}

	subscriptionGroups map[uuid.UUID]*SubscriptionGroup
	routeFn            func(actorID uuid.UUID, msg *Message) error
}

func NewBus() *Bus {
	bus := &Bus{
		inbox:              make(chan any, 100),
		stop:               make(chan struct{}),
		subscriptionGroups: make(map[uuid.UUID]*SubscriptionGroup),
		routeFn:            nil,
	}

	go bus.Start()

	return bus
}

func (b *Bus) Inbox() chan any {
	if b.inbox == nil {
		b.inbox = make(chan any, 100)
	}
	return b.inbox
}

func (b *Bus) Start() {
	go func() {
		for {
			select {
			case msg := <-b.inbox:
				if routeableMsg, ok := msg.(*Message); ok {
					spanID := uuid.New()
					spanCtx := context.WithValue(context.Background(), model.ContextKeySpanID, spanID)
					if err := b.Route(spanCtx, routeableMsg); err != nil {
						ctxLogger := slog.With("span_id", spanID)
						ctxLogger.Error("Failed to route message from bus inbox", "messageID", routeableMsg.GetID(), "error", err)
					}
				}
			case <-b.stop:
				slog.Debug("Bus stopping")
				return
			}
		}
	}()
}

func (b *Bus) Stop() {
	close(b.stop)
}

func (b *Bus) Publish(ctx context.Context, sender uuid.UUID, recipient Recipient, payload any) error {
	msg := &Message{
		ID:              uuid.New(),
		Sender:          sender,
		Payload:         payload,
		Recipient:       recipient,
		ResponseChannel: nil,
	}

	return b.Route(ctx, msg)
}

func (b *Bus) Request(ctx context.Context, sender uuid.UUID, recipient Recipient, payload any) (any, error) {
	spanID := ctx.Value(model.ContextKeySpanID).(uuid.UUID)
	if spanID == uuid.Nil {
		spanID = uuid.New()
		ctx = context.WithValue(ctx, model.ContextKeySpanID, spanID)
	}
	deadlineCtx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	respondTo := make(chan *Message, 1)
	isDone := make(chan struct{})
	var response any

	ctxLogger := slog.With("span_id", spanID)

	ctxLogger.Debug("created response channel")

	go func(c context.Context) {
		select {
		case res := <-respondTo:
			ctxLogger.Debug("Received response to request", "sender", res.GetSender(), "recipient", res.GetRecipient().String(), "responseID", res.GetID(), "responseTo", res.GetResponseTo(), "payload", fmt.Sprintf("%v", res.GetBody()))
			response = res.GetBody()
		case <-c.Done():
			ctxLogger.Warn("Request timed out", "sender", sender, "recipient", recipient.String())
		}
		close(isDone)
	}(deadlineCtx)

	msg := &Message{
		ID:              uuid.New(),
		Sender:          sender,
		Payload:         payload,
		Recipient:       recipient,
		ResponseChannel: respondTo,
	}

	ctxLogger.Debug("created request message", "messageID", msg.GetID(), "sender", sender, "recipient", recipient.String())

	if err := b.Route(ctx, msg); err != nil {
		return nil, err
	}

	ctxLogger.Debug("sent request message", "messageID", msg.GetID(), "sender", sender, "recipient", recipient.String())

	<-isDone

	ctxLogger.Debug("Request handling complete", "sender", sender, "recipient", recipient.String())

	if ctx.Err() == context.DeadlineExceeded {
		return nil, fmt.Errorf("request timed out")
	}

	return response, nil
}

func (b *Bus) Route(ctx context.Context, msg *Message) error {
	for _, group := range b.subscriptionGroups {
		subject := strings.TrimPrefix(msg.GetRecipient().String(), "topic:")
		matches := group.GetPattern().MatchString(subject)
		ctxLogger := slog.With("span_id", ctx.Value(model.ContextKeySpanID))
		ctxLogger.Debug("Routing message", "messageID", msg.GetID(), "recipient", msg.GetRecipient().String(), "subscriptionPattern", group.GetPattern().String(), "matches", matches)
		if matches {
			actors := group.GetActors()
			if b.routeFn != nil {
				for actorID := range actors {
					err := b.routeFn(actorID, msg)
					if err != nil {
						ctxLogger.Error("Failed to route message", "messageID", msg.GetID(), "error", err)
						return err
					}
				}
			} else {
				ctxLogger.Warn("No routing function defined, message will not be delivered", "messageID", msg.GetID())
			}
		}
	}

	return nil
}

func (b *Bus) SetRouteFunction(routeFn func(actorID uuid.UUID, msg *Message) error) {
	b.routeFn = routeFn
}

func (b *Bus) GetSubscriptionGroups() map[uuid.UUID]*SubscriptionGroup {
	return b.subscriptionGroups
}

func (b *Bus) UpdateSubscriptionGroup(group *SubscriptionGroup) {
	if existingGroup, ok := b.subscriptionGroups[group.GetID()]; ok {
		for actorID := range group.GetActors() {
			existingGroup.AddActor(actorID)
		}
		return
	}

	b.subscriptionGroups[group.GetID()] = group
}

func (b *Bus) Subscribe(pattern string, actorID uuid.UUID) (uuid.UUID, error) {
	subscriptionGroup, err := NewSubscriptionGroup(pattern)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to create subscription group: %w", err)
	}

	subscriptionGroup.AddActor(actorID)
	b.UpdateSubscriptionGroup(subscriptionGroup)

	return subscriptionGroup.GetID(), nil
}

func (b *Bus) Unsubscribe(subscriptionID uuid.UUID, actorID uuid.UUID) error {
	if sub, ok := b.subscriptionGroups[subscriptionID]; ok {
		delete(sub.actors, actorID)
		slog.Debug("Unsubscribed actor from topic", "actorID", actorID, "subscriptionID", subscriptionID, "pattern", sub.GetPattern().String())
		if len(sub.actors) == 0 {
			delete(b.subscriptionGroups, subscriptionID)
			slog.Debug("Deleted subscription group as it has no more actors", "subscriptionID", subscriptionID, "pattern", sub.GetPattern().String())
		}
		return nil
	}

	return fmt.Errorf("subscription not found for ID %s and actorID %s", subscriptionID, actorID)
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
