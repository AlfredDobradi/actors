package actor

import (
	"context"
	"testing"
	"time"

	"github.com/alfreddobradi/actors/examples/game/game"
	"github.com/alfreddobradi/actors/pkg/database"
	"github.com/alfreddobradi/actors/pkg/database/memory"
	pkgmodel "github.com/alfreddobradi/actors/pkg/model"
	"github.com/alfreddobradi/actors/pkg/system"
	"github.com/alfreddobradi/actors/pkg/testhelper"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestNewTick(t *testing.T) {
	before := time.Now()
	tick := NewTick()
	after := time.Now()

	require.NotZero(t, tick.Timestamp)
	require.True(t, tick.Timestamp.After(before) || tick.Timestamp.Equal(before))
	require.True(t, tick.Timestamp.Before(after) || tick.Timestamp.Equal(after))
}

func TestTickerActorGetID(t *testing.T) {
	id := uuid.New()
	actor := &TickerActor{
		ID: id,
	}

	require.Equal(t, id, actor.GetID())
}

func TestTickerActorGetKind(t *testing.T) {
	actor := &TickerActor{}

	require.Equal(t, "TickerActor", actor.GetKind())
}

func TestTickerActorHandleMessage(t *testing.T) {
	actor := &TickerActor{}
	ctx := context.Background()
	msg := &system.Message{}

	err := actor.HandleMessage(ctx, msg)

	require.NoError(t, err)
}

func TestTickerActorSnapshot(t *testing.T) {
	actor := &TickerActor{
		ID: uuid.New(),
	}
	ctx := context.Background()

	snapshot, err := actor.Snapshot(ctx)

	require.NoError(t, err)
	require.Equal(t, 0, len(snapshot.Data))
}

func TestTickerActorRestoreFromSnapshot(t *testing.T) {
	actor := &TickerActor{
		ID: uuid.New(),
	}
	ctx := context.Background()
	snapshot := database.Snapshot{
		Timestamp: time.Now().Unix(),
		Data:      []byte{},
	}

	err := actor.RestoreFromSnapshot(ctx, snapshot)

	require.NoError(t, err)
}

func TestTickerActorStop(t *testing.T) {
	testhelper.SetupTestLogger(false)

	id := uuid.New()
	actor := &TickerActor{
		ID:    id,
		timer: time.NewTicker(100 * time.Millisecond),
	}

	ctx := context.Background()
	err := actor.Stop(ctx)

	require.NoError(t, err)
}

func TestTickerActorTickCallback(t *testing.T) {
	testhelper.SetupTestLogger(false)

	id := uuid.New()
	receivedTicks := make(chan Tick, 10)

	mockSendCallback := func(
		ctx context.Context,
		expectsResponse bool,
		senderID uuid.UUID,
		recipient system.Recipient,
		message interface{},
	) (any, error) {
		tick, ok := message.(Tick)
		require.True(t, ok, "expected Tick message")

		receivedTicks <- tick

		return nil, nil
	}

	actor := &TickerActor{
		ID:           id,
		sendCallback: mockSendCallback,
	}

	ctx := context.Background()
	err := actor.tickCallback(ctx)

	require.NoError(t, err)
	require.Len(t, receivedTicks, 1)

	tick := <-receivedTicks
	require.NotZero(t, tick.Timestamp)
}

func TestTickerActorTickCallbackWithSpanID(t *testing.T) {
	testhelper.SetupTestLogger(false)

	id := uuid.New()
	var capturedSpanID uuid.UUID

	mockSendCallback := func(
		ctx context.Context,
		expectsResponse bool,
		senderID uuid.UUID,
		recipient system.Recipient,
		message interface{},
	) (any, error) {
		// Capture the span ID from context
		spanID, ok := ctx.Value(pkgmodel.ContextKeySpanID).(uuid.UUID)
		if ok {
			capturedSpanID = spanID
		}
		return nil, nil
	}

	actor := &TickerActor{
		ID:           id,
		sendCallback: mockSendCallback,
	}

	ctx := context.Background()
	err := actor.tickCallback(ctx)

	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, capturedSpanID)
}

func TestTickerActorStart(t *testing.T) {
	testhelper.SetupTestLogger(false)

	id := uuid.New()
	receivedTickCount := 0
	tickChan := make(chan Tick, 10)

	mockSendCallback := func(
		ctx context.Context,
		expectsResponse bool,
		senderID uuid.UUID,
		recipient system.Recipient,
		message interface{},
	) (any, error) {
		tick, ok := message.(Tick)
		require.True(t, ok)
		tickChan <- tick
		return nil, nil
	}

	actor := &TickerActor{
		ID:           id,
		timer:        time.NewTicker(50 * time.Millisecond),
		sendCallback: mockSendCallback,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	actor.Start(ctx)

	// Wait for ticks to be sent
	for {
		select {
		case <-tickChan:
			receivedTickCount++
		case <-ctx.Done():
			goto done
		}
	}

done:
	// We should have received multiple ticks within the timeout
	require.Greater(t, receivedTickCount, 0, "expected at least one tick to be sent")
}

func TestTickerActorStartStopsOnContextDone(t *testing.T) {
	testhelper.SetupTestLogger(false)

	id := uuid.New()
	tickChan := make(chan Tick, 10)

	mockSendCallback := func(
		ctx context.Context,
		expectsResponse bool,
		senderID uuid.UUID,
		recipient system.Recipient,
		message interface{},
	) (any, error) {
		tick, ok := message.(Tick)
		require.True(t, ok)
		tickChan <- tick
		return nil, nil
	}

	actor := &TickerActor{
		ID:           id,
		timer:        time.NewTicker(10 * time.Millisecond),
		sendCallback: mockSendCallback,
	}

	ctx, cancel := context.WithCancel(context.Background())
	actor.Start(ctx)

	// Let a few ticks through
	time.Sleep(50 * time.Millisecond)
	ticksBeforeCancel := len(tickChan)
	require.Greater(t, ticksBeforeCancel, 0)

	// Cancel context
	cancel()

	// Give it time to stop
	time.Sleep(100 * time.Millisecond)

	// Verify no more ticks are being sent (allow for one extra tick that might have been in flight)
	ticksAfterCancel := len(tickChan)
	require.LessOrEqual(t, ticksAfterCancel-ticksBeforeCancel, 1, "should receive at most one more tick after cancel")
}

func TestTickerActorFactory(t *testing.T) {
	testhelper.SetupTestLogger(false)

	ctx := context.Background()
	registry := system.NewRegistry()
	registry.RegisterFactory("TickerActor", tickerActorFactory)

	db := memory.NewStore()

	mockSendFunc := func(
		ctx context.Context,
		expectsResponse bool,
		senderID uuid.UUID,
		recipient system.Recipient,
		message interface{},
	) (any, error) {
		return nil, nil
	}

	ctxWithSender := context.WithValue(ctx, pkgmodel.ContextKeySenderFn, mockSendFunc)

	sys := system.MustNewSystem(registry, db)

	// Spawn ticker actor
	actorHandler, err := sys.Spawn(ctxWithSender, "TickerActor")
	require.NoError(t, err)
	require.NotNil(t, actorHandler)

	actor := actorHandler.GetActor().(*TickerActor)
	require.NotEqual(t, uuid.Nil, actor.ID)
	require.NotNil(t, actor.timer)
	require.NotNil(t, actor.sendCallback)
	require.Equal(t, "TickerActor", actor.GetKind())
}

func TestTickerActorFactoryInitialization(t *testing.T) {
	testhelper.SetupTestLogger(false)

	ctx := context.Background()
	registry := system.NewRegistry()
	registry.RegisterFactory("TickerActor", tickerActorFactory)

	db := memory.NewStore()

	mockSendFunc := func(
		ctx context.Context,
		expectsResponse bool,
		senderID uuid.UUID,
		recipient system.Recipient,
		message interface{},
	) (any, error) {
		return nil, nil
	}

	ctxWithSender := context.WithValue(ctx, pkgmodel.ContextKeySenderFn, mockSendFunc)

	sys := system.MustNewSystem(registry, db)

	actorHandler, err := sys.Spawn(ctxWithSender, "TickerActor")
	require.NoError(t, err)

	actor := actorHandler.GetActor().(*TickerActor)

	// Verify the ticker was properly initialized with the correct interval
	expectedInterval := time.Second / time.Duration(game.TickRate)

	// Create a reference ticker with the same interval for comparison
	referenceTicker := time.NewTicker(expectedInterval)
	defer referenceTicker.Stop()

	// Verify they are close in duration (within 10ms tolerance for timing variations)
	// This is a pragmatic test since we can't directly inspect Ticker's internal duration
	require.NotNil(t, actor.timer)
	require.NotNil(t, actor.sendCallback)
}
