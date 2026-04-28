/*
This example demonstrates the simplest usage of the actor system.

TODO list:
* Keep track of actors in the registry (and also remotely in etcd)
* Add a way for actors to recover after errors
* Add support for distributed actors across multiple nodes
* Make actors directly addressable both locally and remotely
* Add support for actor supervision and monitoring
* Add support for actor state persistence and recovery
*/
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/alfreddobradi/actors/pkg/system"
	"github.com/davecgh/go-spew/spew"
	"github.com/google/uuid"
)

func persistentActorID() uuid.UUID {
	// In a real implementation, this would likely be generated based on some stable identifier or retrieved from a database to ensure it remains consistent across restarts.
	return uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
}

// ExampleActor is a simple implementation of the Actor interface for the example
type ExampleActor struct {
	ID uuid.UUID
}

func (e *ExampleActor) GetID() uuid.UUID {
	return e.ID
}

func (e *ExampleActor) GetKind() string {
	return "ExampleActor"
}

func (e *ExampleActor) Start(ctx context.Context) {}

func (e *ExampleActor) Stop(ctx context.Context) error {
	return nil
}

func (e *ExampleActor) HandleMessage(ctx context.Context, msg system.Message) system.HandleError {
	spew.Dump(msg)
	slog.Info("Handling message", "actorID", e.GetID(), "messageID", msg.ID, "payload", string(msg.Payload))

	return nil
}

// exampleActorFactory is the factory method for creating instances of ExampleActor
func exampleActorFactory(_ context.Context) system.Actor {
	return &ExampleActor{
		ID: persistentActorID(),
	}
}

// Hook implementations for demo purposes, simple logging
func preStartHook(ctx context.Context, actor system.Actor) error {
	slog.Info("Running pre-start hook", "actorID", actor.GetID())
	return nil
}

func postStartHook(ctx context.Context, actor system.Actor) error {
	slog.Info("Running post-start hook", "actorID", actor.GetID())
	return nil
}

func poisonedHook(ctx context.Context, actor system.Actor) error {
	slog.Info("Running poisoned hook", "actorID", actor.GetID())
	return nil
}

func terminatedHook(ctx context.Context, actor system.Actor) error {
	slog.Info("Running terminated hook", "actorID", actor.GetID())
	return nil
}

func crashHook(ctx context.Context, actor system.Actor) error {
	slog.Info("Running crash hook", "actorID", actor.GetID())
	return nil
}

func spawnActor(sys *system.System) (*system.ActorHandler, error) {
	ctx := context.Background()
	return sys.Spawn(
		ctx,
		"ExampleActor",
		system.WithPreStartHook(preStartHook),
		system.WithPostStartHook(postStartHook),
		system.WithPoisonedHook(poisonedHook),
		system.WithTerminatedHook(terminatedHook),
		system.WithCrashHook(crashHook),
	)
}

func main() {
	ctx := context.Background()

	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})))

	registry := system.NewRegistry()

	// Register the ExampleActor factory with the registry
	registry.RegisterFactory("ExampleActor", exampleActorFactory)

	sys := system.NewSystem(nil, registry)

	// Spawn an instance of ExampleActor using the factory function stored in the registry.
	// This is also where we can attach hooks to the actor's lifecycle events.
	handlerFoo, err := spawnActor(sys)
	if err != nil {
		slog.Error("Failed to spawn actor", "error", err)
		os.Exit(1)
	}

	// Simulate trying to spawn another actor with the same ID to demonstrate that we return the existing handler instead of creating a new actor.
	handlerBar, err := spawnActor(sys)
	if err != nil {
		slog.Error("Failed to spawn actor", "error", err)
		os.Exit(1)
	}

	slog.Debug("actor spawned", "foo_address", fmt.Sprintf("%p", handlerFoo), "bar_address", fmt.Sprintf("%p", handlerBar))

	// Send more messages than it will (probably) process to demonstrate the poisoned state (draining the channel and logging the messages we didn't process) and hooks
	for range 100 {
		handlerFoo.SendMessage(ctx, system.Message{ID: uuid.New(), Payload: []byte("hi")})
	}

	handlerFoo.Stop()

	time.Sleep(200 * time.Millisecond)
	// Send another message after stopping to demonstrate that it won't be processed.
	handlerFoo.SendMessage(ctx, system.Message{ID: uuid.New(), Payload: []byte("this should not be handled")})

	handlerFoo.WaitForTermination()
}
