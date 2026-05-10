package system_test

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"sync"
	"testing"
	"time"

	"github.com/alfreddobradi/actors/pkg/database"
	"github.com/alfreddobradi/actors/pkg/database/memory"
	"github.com/alfreddobradi/actors/pkg/system"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

const mockBusActorKind = "MockBusActor"

type MockBusActor struct {
	ID       uuid.UUID
	mx       *sync.Mutex
	messages []string
}

func (a *MockBusActor) GetID() uuid.UUID               { return a.ID }
func (a *MockBusActor) GetKind() string                { return "MockBusActor" }
func (a *MockBusActor) Start(ctx context.Context)      {}
func (a *MockBusActor) Stop(ctx context.Context) error { return nil }
func (a *MockBusActor) Snapshot(ctx context.Context) (database.Snapshot, error) {
	return database.Snapshot{}, nil
}
func (a *MockBusActor) RestoreFromSnapshot(ctx context.Context, snapshot database.Snapshot) error {
	return nil
}
func (a *MockBusActor) HandleMessage(ctx context.Context, msg *system.Message) system.HandleError {
	a.mx.Lock()
	slog.Debug("MockBusActor handling message", "actorID", a.GetID(), "messageID", msg.GetID(), "payload", fmt.Sprintf("%v", msg.GetBody()))
	a.messages = append(a.messages, fmt.Sprintf("%v", msg.GetBody()))
	a.mx.Unlock()

	return nil
}

func mockBusActorFactory(ctx context.Context) system.Actor {
	return &MockBusActor{
		ID:       uuid.New(),
		mx:       &sync.Mutex{},
		messages: make([]string, 0),
	}
}

func TestRouting(t *testing.T) {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	registry := system.NewRegistry()
	registry.RegisterFactory(mockBusActorKind, mockBusActorFactory)

	db, err := memory.NewStore()
	require.NoError(t, err)
	sys := system.MustNewSystem(registry, db)

	handlerFoo, err := sys.Spawn(context.Background(), mockBusActorKind)
	if err != nil {
		t.Fatalf("Failed to spawn actor: %v", err)
	}
	handlerBar, err := sys.Spawn(context.Background(), mockBusActorKind)
	if err != nil {
		t.Fatalf("Failed to spawn actor: %v", err)
	}

	_, err = sys.Subscribe("^foo", handlerFoo.GetActor().GetID())
	if err != nil {
		t.Fatalf("Failed to subscribe actor: %v", err)
	}
	_, err = sys.Subscribe("bar$", handlerBar.GetActor().GetID())
	if err != nil {
		t.Fatalf("Failed to subscribe actor: %v", err)
	}

	msgFoo := &system.Message{
		ID:        uuid.New(),
		Payload:   "foo",
		Recipient: system.Recipient{Kind: system.RecipientKindTopic, Subject: "foo"},
	}

	msgBar := &system.Message{
		ID:        uuid.New(),
		Payload:   "bar",
		Recipient: system.Recipient{Kind: system.RecipientKindTopic, Subject: "bar"},
	}

	msgFoobar := &system.Message{
		ID:        uuid.New(),
		Payload:   []byte("foobar"),
		Recipient: system.Recipient{Kind: system.RecipientKindTopic, Subject: "foobar"},
	}

	wg := &sync.WaitGroup{}
	tmx := &sync.Mutex{}
	times := []time.Duration{}

	cmx := &sync.Mutex{}
	counts := make(map[string]int)
	start := time.Now()
	msgs := []*system.Message{msgFoo, msgBar, msgFoobar}

	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			which := i % 3

			start := time.Now()
			require.NoError(t, sys.Route(context.Background(), msgs[which]))
			tmx.Lock()
			times = append(times, time.Since(start))
			tmx.Unlock()

			cmx.Lock()
			counts[msgs[which].Recipient.Subject]++
			cmx.Unlock()
		}(i)
	}
	wg.Wait()

	sumTime := time.Duration(0)
	for _, t := range times {
		sumTime += t
	}
	avgTime := sumTime / time.Duration(len(times))
	slog.Debug("Average routing time", "avgTime", avgTime.String())
	slog.Debug("Total loop time", "totalTime", time.Since(start).String())

	time.Sleep(1 * time.Millisecond) // Wait for messages to be processed

	// Check that the correct messages were received by each actor
	mockActorFoo := handlerFoo.GetActor().(*MockBusActor)
	mockActorBar := handlerBar.GetActor().(*MockBusActor)

	require.NotContains(t, mockActorFoo.messages, "bar")
	require.NotContains(t, mockActorBar.messages, "foo")

	require.Equal(t, len(mockActorFoo.messages), counts["foo"]+counts["foobar"])
	require.Equal(t, len(mockActorBar.messages), counts["bar"]+counts["foobar"])

	// Clean up
	handlerFoo.Stop()
	handlerBar.Stop()

	handlerFoo.WaitForTermination()
	handlerBar.WaitForTermination()
}

func TestSubscribe(t *testing.T) {
	bus := system.NewBus()

	fooID := uuid.New()
	barID := uuid.New()

	groupID1, err := bus.Subscribe("^foo", fooID)
	require.NoError(t, err)

	groupID2, err := bus.Subscribe("^foo", barID)
	require.NoError(t, err)

	require.Equal(t, groupID1, groupID2)

	groupID3, err := bus.Subscribe("bar$", barID)
	require.NoError(t, err)

	require.NotEqual(t, groupID1, groupID3)

	groups := bus.GetSubscriptionGroups()
	require.Len(t, groups, 2)

	group1, ok := groups[groupID1]
	require.True(t, ok)
	require.Equal(t, "^foo", group1.GetPattern().String())
	require.Contains(t, group1.GetActors(), fooID)
	require.Contains(t, group1.GetActors(), barID)

	group2, ok := groups[groupID3]
	require.True(t, ok)
	require.Equal(t, "bar$", group2.GetPattern().String())
	require.Contains(t, group2.GetActors(), barID)
}

func TestUnsubscribe(t *testing.T) {
	bus := system.NewBus()

	fooID := uuid.New()
	barID := uuid.New()

	groupID1, err := bus.Subscribe("^foo", fooID)
	require.NoError(t, err)

	_, err = bus.Subscribe("^foo", barID)
	require.NoError(t, err)

	err = bus.Unsubscribe(groupID1, fooID)
	require.NoError(t, err)

	groups := bus.GetSubscriptionGroups()
	require.Len(t, groups, 1)

	group1, ok := groups[groupID1]
	require.True(t, ok)
	require.Equal(t, "^foo", group1.GetPattern().String())
	require.NotContains(t, group1.GetActors(), fooID)
	require.Contains(t, group1.GetActors(), barID)

	err = bus.Unsubscribe(groupID1, barID)
	require.NoError(t, err)

	groups = bus.GetSubscriptionGroups()
	require.Len(t, groups, 0)
}

func generateSubscriptions(b *testing.B, n int) []*system.Subscription {
	patternText := []string{
		"foo",
		"bar",
		"baz",
		"qux",
		"quux",
	}
	subs := make([]*system.Subscription, n)
	for i := 0; i < n; i++ {
		text := patternText[i%5]
		sub, err := system.NewSubscription(text, uuid.New())
		require.NoError(b, err)
		subs[i] = sub
	}
	return subs
}

func BenchmarkRoutingSingular(b *testing.B) {
	n := 1000
	subs := generateSubscriptions(b, n)

	topic := "foo"

	for i := 0; i < b.N; i++ {
		for _, sub := range subs {
			if sub.GetPattern().MatchString(topic) {
				slog.Debug("matched subscription", "pattern", sub.GetPattern().String())
				continue
			}
		}
	}
}

func BenchmarkRoutingGrouped(b *testing.B) {
	n := 1000
	subs := generateSubscriptions(b, n)

	subsGrouped := make(map[string][]*system.Subscription)
	for _, sub := range subs {
		subsGrouped[sub.GetPattern().String()] = append(subsGrouped[sub.GetPattern().String()], sub)
	}

	topic := "foo"

	for i := 0; i < b.N; i++ {
		for patternStr := range subsGrouped {
			pattern, err := regexp.Compile(patternStr)
			require.NoError(b, err)
			if pattern.MatchString(topic) {
				slog.Debug("matched subscription", "pattern", pattern.String())
				continue
			}
		}
	}
}
