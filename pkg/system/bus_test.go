package system_test

import (
	"context"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/alfreddobradi/actors/pkg/system"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

const mockBusActorKind = "MockBusActor"

type MockMessage struct {
	ID      uuid.UUID
	Payload []byte
	Topic   string
}

func (m *MockMessage) GetID() uuid.UUID { return m.ID }
func (m *MockMessage) GetBody() []byte  { return m.Payload }
func (m *MockMessage) GetTopic() string { return m.Topic }

type MockBusActor struct {
	ID       uuid.UUID
	mx       *sync.Mutex
	messages []string
}

func (a *MockBusActor) GetID() uuid.UUID               { return a.ID }
func (a *MockBusActor) GetKind() string                { return "MockBusActor" }
func (a *MockBusActor) Start(ctx context.Context)      {}
func (a *MockBusActor) Stop(ctx context.Context) error { return nil }
func (a *MockBusActor) HandleMessage(ctx context.Context, msg system.Message) system.HandleError {
	a.mx.Lock()
	slog.Debug("MockBusActor handling message", "actorID", a.GetID(), "messageID", msg.ID, "payload", string(msg.Payload))
	a.messages = append(a.messages, string(msg.Payload))
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

	bus := system.NewBus()
	sys := system.NewSystem(bus, registry)

	handlerFoo, err := sys.Spawn(context.Background(), mockBusActorKind)
	if err != nil {
		t.Fatalf("Failed to spawn actor: %v", err)
	}
	handlerBar, err := sys.Spawn(context.Background(), mockBusActorKind)
	if err != nil {
		t.Fatalf("Failed to spawn actor: %v", err)
	}

	err = sys.Subscribe("^foo", handlerFoo.GetActor().GetID())
	if err != nil {
		t.Fatalf("Failed to subscribe actor: %v", err)
	}
	err = sys.Subscribe("bar$", handlerBar.GetActor().GetID())
	if err != nil {
		t.Fatalf("Failed to subscribe actor: %v", err)
	}

	msgFoo := &MockMessage{
		ID:      uuid.New(),
		Payload: []byte("foo"),
		Topic:   "foo",
	}

	msgBar := &MockMessage{
		ID:      uuid.New(),
		Payload: []byte("bar"),
		Topic:   "bar",
	}

	msgFoobar := &MockMessage{
		ID:      uuid.New(),
		Payload: []byte("foobar"),
		Topic:   "foobar",
	}

	wg := &sync.WaitGroup{}
	tmx := &sync.Mutex{}
	times := []time.Duration{}

	cmx := &sync.Mutex{}
	counts := make(map[string]int)
	start := time.Now()
	msgs := []*MockMessage{msgFoo, msgBar, msgFoobar}

	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			which := i % 3

			start := time.Now()
			require.NoError(t, sys.Route(msgs[which]))
			tmx.Lock()
			times = append(times, time.Since(start))
			tmx.Unlock()

			cmx.Lock()
			counts[msgs[which].Topic]++
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
