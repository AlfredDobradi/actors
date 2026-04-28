package system_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/alfreddobradi/actors/pkg/system"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// MockActor is used for testing.
type MockActor struct {
	id           uuid.UUID
	kind         string
	messageCount int
	messages     [][]byte
	mu           sync.Mutex
	// Control behavior
	shouldCrash      bool
	crashAfterCount  int
	returnsRecovable bool
}

func NewMockActor(kind string) *MockActor {
	return &MockActor{
		id:               uuid.New(),
		kind:             kind,
		returnsRecovable: true,
	}
}

func (m *MockActor) GetID() uuid.UUID {
	return m.id
}

func (m *MockActor) GetKind() string {
	return m.kind
}

func (m *MockActor) Start(ctx context.Context) {}

func (m *MockActor) Stop(ctx context.Context) error {
	return nil
}

func (m *MockActor) HandleMessage(ctx context.Context, msg system.Message) system.HandleError {
	m.mu.Lock()
	m.messageCount++
	m.messages = append(m.messages, msg.Payload)
	count := m.messageCount
	m.mu.Unlock()

	// Simulate crash after N messages
	if m.shouldCrash && count >= m.crashAfterCount {
		return &MockError{
			msg:         "intentional crash",
			recoverable: m.returnsRecovable,
		}
	}

	return &MockError{
		msg:         "ok",
		recoverable: true,
	}
}

type MockError struct {
	msg         string
	recoverable bool
}

func (me *MockError) IsRecoverable() bool {
	return me.recoverable
}

func (me *MockError) Error() string {
	return me.msg
}

// HookTracker records the order and timing of hook executions.
type HookTracker struct {
	mu    sync.Mutex
	order []string
}

func (ht *HookTracker) Record(hookName string) {
	ht.mu.Lock()
	defer ht.mu.Unlock()
	ht.order = append(ht.order, hookName)
}

func (ht *HookTracker) GetOrder() []string {
	ht.mu.Lock()
	defer ht.mu.Unlock()
	// Make a copy
	result := make([]string, len(ht.order))
	copy(result, ht.order)
	return result
}

func (ht *HookTracker) CreateHook(hookName string) system.Hook {
	return func(ctx context.Context, actor system.Actor) error {
		ht.Record(hookName)
		return nil
	}
}

// TestPreStartHookExecution verifies preStart hooks run before the actor starts.
func TestPreStartHookExecution(t *testing.T) {
	tracker := &HookTracker{}
	ctx := context.Background()

	actor := NewMockActor("test")
	registry := system.NewRegistry()
	registry.RegisterFactory(actor.GetKind(), func(ctx context.Context) system.Actor { return actor })
	sys := system.NewSystem(nil, registry)

	handler, err := sys.Spawn(
		ctx,
		actor.GetKind(),
		system.WithPreStartHook(tracker.CreateHook("preStart")),
	)
	require.NoError(t, err)

	// Give goroutine time to start
	time.Sleep(100 * time.Millisecond)
	handler.Stop()
	handler.WaitForTermination()

	order := tracker.GetOrder()
	if len(order) == 0 {
		t.Fatal("no hooks executed")
	}
	if order[0] != "preStart" {
		t.Errorf("expected preStart to be first, got %v", order)
	}
}

// TestPostStartHookExecution verifies postStart hooks run after the actor starts.
func TestPostStartHookExecution(t *testing.T) {
	tracker := &HookTracker{}
	ctx := context.Background()

	actor := NewMockActor("test")
	registry := system.NewRegistry()
	registry.RegisterFactory(actor.GetKind(), func(ctx context.Context) system.Actor { return actor })
	sys := system.NewSystem(nil, registry)

	handler, err := sys.Spawn(
		ctx,
		actor.GetKind(),
		system.WithPostStartHook(tracker.CreateHook("postStart")),
	)
	require.NoError(t, err)

	// Give goroutine time to start
	time.Sleep(100 * time.Millisecond)
	handler.Stop()
	handler.WaitForTermination()

	order := tracker.GetOrder()
	found := false
	for _, h := range order {
		if h == "postStart" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("postStart hook never executed, got order: %v", order)
	}
}

// TestHookExecutionOrder verifies hooks execute in the correct order during normal termination.
func TestHookExecutionOrder(t *testing.T) {
	tracker := &HookTracker{}
	ctx := context.Background()

	actor := NewMockActor("test")
	registry := system.NewRegistry()
	registry.RegisterFactory(actor.GetKind(), func(ctx context.Context) system.Actor { return actor })
	sys := system.NewSystem(nil, registry)

	handler, err := sys.Spawn(
		ctx,
		actor.GetKind(),
		system.WithPreStartHook(tracker.CreateHook("preStart")),
		system.WithPostStartHook(tracker.CreateHook("postStart")),
		system.WithPoisonedHook(tracker.CreateHook("poisoned")),
		system.WithTerminatedHook(tracker.CreateHook("terminated")),
	)
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)
	handler.Stop()
	handler.WaitForTermination()

	order := tracker.GetOrder()

	// Verify all expected hooks were executed
	expectedHooks := map[string]bool{"preStart": false, "postStart": false, "poisoned": false, "terminated": false}
	for _, h := range order {
		if _, exists := expectedHooks[h]; exists {
			expectedHooks[h] = true
		}
	}

	for hook, executed := range expectedHooks {
		if !executed {
			t.Errorf("expected hook %q was not executed, order: %v", hook, order)
		}
	}

	// Verify poisoned comes before terminated
	poisonedIdx := -1
	terminatedIdx := -1
	for i, h := range order {
		if h == "poisoned" {
			poisonedIdx = i
		}
		if h == "terminated" {
			terminatedIdx = i
		}
	}
	if poisonedIdx >= 0 && terminatedIdx >= 0 && poisonedIdx >= terminatedIdx {
		t.Errorf("expected poisoned before terminated, got order: %v", order)
	}
}

// TestPoisonedHookExecution verifies poisoned hooks run when Stop() is called.
func TestPoisonedHookExecution(t *testing.T) {
	tracker := &HookTracker{}
	ctx := context.Background()

	actor := NewMockActor("test")
	registry := system.NewRegistry()
	registry.RegisterFactory(actor.GetKind(), func(ctx context.Context) system.Actor { return actor })
	sys := system.NewSystem(nil, registry)

	handler, err := sys.Spawn(
		ctx,
		actor.GetKind(),
		system.WithPoisonedHook(tracker.CreateHook("poisoned")),
		system.WithTerminatedHook(tracker.CreateHook("terminated")),
	)
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)
	handler.Stop()
	handler.WaitForTermination()

	order := tracker.GetOrder()
	found := false
	for _, h := range order {
		if h == "poisoned" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("poisoned hook was not executed, order: %v", order)
	}
}

// TestTerminatedHookExecution verifies terminated hooks always run on termination.
func TestTerminatedHookExecution(t *testing.T) {
	tracker := &HookTracker{}
	ctx := context.Background()

	actor := NewMockActor("test")
	registry := system.NewRegistry()
	registry.RegisterFactory(actor.GetKind(), func(ctx context.Context) system.Actor { return actor })
	sys := system.NewSystem(nil, registry)

	handler, err := sys.Spawn(
		ctx,
		actor.GetKind(),
		system.WithTerminatedHook(tracker.CreateHook("terminated")),
	)
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)
	handler.Stop()
	handler.WaitForTermination()

	order := tracker.GetOrder()
	found := false
	for _, h := range order {
		if h == "terminated" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("terminated hook was not executed, order: %v", order)
	}
}

// TestCrashHookExecution verifies crash hooks run on unexpected termination.
func TestCrashHookExecution(t *testing.T) {
	tracker := &HookTracker{}
	ctx := context.Background()

	actor := NewMockActor("test")
	actor.shouldCrash = true
	actor.crashAfterCount = 0      // Crash on first message
	actor.returnsRecovable = false // Make the crash non-recoverable

	registry := system.NewRegistry()
	registry.RegisterFactory(actor.GetKind(), func(ctx context.Context) system.Actor { return actor })
	sys := system.NewSystem(nil, registry)

	handler, err := sys.Spawn(
		ctx,
		actor.GetKind(),
		system.WithCrashHook(tracker.CreateHook("crash")),
		system.WithTerminatedHook(tracker.CreateHook("terminated")),
	)
	require.NoError(t, err)

	// Send a message to trigger the crash
	msg := system.Message{
		ID:      uuid.New(),
		Payload: []byte("test"),
	}
	handler.SendMessage(ctx, msg)

	// Wait for the actor to crash and hooks to execute
	time.Sleep(100 * time.Millisecond)
	handler.WaitForTermination()

	order := tracker.GetOrder()
	found := false
	for _, h := range order {
		if h == "crash" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("crash hook was not executed, order: %v", order)
	}
}

// TestCrashHookNotExecutedOnNormalStop verifies crash hook is NOT run on normal termination.
func TestCrashHookNotExecutedOnNormalStop(t *testing.T) {
	tracker := &HookTracker{}
	ctx := context.Background()

	actor := NewMockActor("test")
	registry := system.NewRegistry()
	registry.RegisterFactory(actor.GetKind(), func(ctx context.Context) system.Actor { return actor })
	sys := system.NewSystem(nil, registry)

	handler, err := sys.Spawn(
		ctx,
		actor.GetKind(),
		system.WithCrashHook(tracker.CreateHook("crash")),
		system.WithPoisonedHook(tracker.CreateHook("poisoned")),
		system.WithTerminatedHook(tracker.CreateHook("terminated")),
	)
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)
	handler.Stop()
	handler.WaitForTermination()

	order := tracker.GetOrder()
	for _, h := range order {
		if h == "crash" {
			t.Errorf("crash hook should not execute on normal stop, order: %v", order)
		}
	}
}

// TestMultipleHooksOfSameType verifies multiple hooks of the same type all execute.
func TestMultipleHooksOfSameType(t *testing.T) {
	tracker := &HookTracker{}
	ctx := context.Background()

	actor := NewMockActor("test")
	registry := system.NewRegistry()
	registry.RegisterFactory(actor.GetKind(), func(ctx context.Context) system.Actor { return actor })
	sys := system.NewSystem(nil, registry)

	handler, err := sys.Spawn(
		ctx,
		actor.GetKind(),
		system.WithTerminatedHook(tracker.CreateHook("terminated1")),
		system.WithTerminatedHook(tracker.CreateHook("terminated2")),
		system.WithTerminatedHook(tracker.CreateHook("terminated3")),
	)
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)
	handler.Stop()
	handler.WaitForTermination()

	order := tracker.GetOrder()
	count := 0
	for _, h := range order {
		if h == "terminated1" || h == "terminated2" || h == "terminated3" {
			count++
		}
	}
	if count != 3 {
		t.Errorf("expected 3 terminated hooks to execute, got %d: %v", count, order)
	}
}

// TestMessageProcessingBeforeCrash verifies messages are processed before crash detection.
func TestMessageProcessingBeforeCrash(t *testing.T) {
	ctx := context.Background()
	actor := NewMockActor("test")
	actor.shouldCrash = true
	actor.crashAfterCount = 3      // Crash after 3 messages
	actor.returnsRecovable = false // Make the crash non-recoverable

	registry := system.NewRegistry()
	registry.RegisterFactory(actor.GetKind(), func(ctx context.Context) system.Actor { return actor })
	sys := system.NewSystem(nil, registry)

	handler, err := sys.Spawn(
		ctx,
		actor.GetKind(),
	)
	require.NoError(t, err)

	// Send 2 messages (should be processed normally)
	for i := 0; i < 2; i++ {
		handler.SendMessage(ctx, system.Message{
			ID:      uuid.New(),
			Payload: []byte("msg"),
		})
	}

	time.Sleep(100 * time.Millisecond)

	if actor.messageCount != 2 {
		t.Errorf("expected 2 messages to be processed, got %d", actor.messageCount)
	}

	// Send message that triggers crash
	handler.SendMessage(ctx, system.Message{
		ID:      uuid.New(),
		Payload: []byte("crash"),
	})

	time.Sleep(100 * time.Millisecond)
	handler.WaitForTermination()

	if actor.messageCount < 3 {
		t.Errorf("expected at least 3 messages to be processed, got %d", actor.messageCount)
	}
}
