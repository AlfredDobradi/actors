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
	"log/slog"
	"os"
	"time"

	"github.com/alfreddobradi/actors/pkg/system"
	"github.com/davecgh/go-spew/spew"
	"github.com/google/uuid"
)

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
func exampleActorFactory() system.Actor {
	return &ExampleActor{
		ID: uuid.New(),
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

func main() {
	ctx := context.Background()

	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})))

	registry := system.NewRegistry()

	// Register the ExampleActor factory with the registry
	registry.RegisterFactory("ExampleActor", func(context.Context) system.Actor {
		return exampleActorFactory()
	})

	sys := system.NewSystem(nil, registry)

	// Spawn an instance of ExampleActor using the factory function stored in the registry.
	// This is also where we can attach hooks to the actor's lifecycle events.
	handler, err := sys.Spawn(
		ctx,
		"ExampleActor",
		system.WithPreStartHook(preStartHook),
		system.WithPostStartHook(postStartHook),
		system.WithPoisonedHook(poisonedHook),
		system.WithTerminatedHook(terminatedHook),
		system.WithCrashHook(crashHook),
	)
	if err != nil {
		slog.Error("Failed to spawn actor", "error", err)
		os.Exit(1)
	}

	// Send more messages than it will (probably) process to demonstrate the poisoned state (draining the channel and logging the messages we didn't process) and hooks
	for range 100 {
		handler.SendMessage(ctx, system.Message{ID: uuid.New(), Payload: []byte("hi")})
	}

	handler.Stop()

	time.Sleep(200 * time.Millisecond)
	// Send another message after stopping to demonstrate that it won't be processed.
	handler.SendMessage(ctx, system.Message{ID: uuid.New(), Payload: []byte("this should not be handled")})

	handler.WaitForTermination()
}
