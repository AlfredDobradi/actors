package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alfreddobradi/actors/examples/game/actor"
	"github.com/alfreddobradi/actors/examples/game/api"
	"github.com/alfreddobradi/actors/examples/game/database"
	"github.com/alfreddobradi/actors/examples/game/database/mmap"
	"github.com/alfreddobradi/actors/examples/game/logging"
	"github.com/alfreddobradi/actors/pkg/system"
	"github.com/joho/godotenv"
)

const (
	topicCharacter = "character"
	topicTicks     = "ticks"
)

func main() {
	godotenv.Load()
	logging.Init()

	registry := system.NewRegistry()
	actor.InitFactories(registry)

	var (
		db    database.DB
		dbErr error
	)
	if db, dbErr = mmap.NewStore("game_data.json"); dbErr != nil {
		slog.Error("Failed to initialize database", "error", dbErr)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sys := system.NewSystem("game_example", registry)

	server := api.NewServer(":8080", sys, db)
	go server.Start()

	characterStore, err := sys.Spawn(ctx, "CharacterStore",
		system.WithSubscription(topicTicks),
		system.WithSubscription(topicCharacter),
	)
	if err != nil {
		slog.Error("Failed to spawn actor", "error", err)
		return
	}

	// if err := sys.Publish(ctx, uuid.Nil, system.Recipient{Kind: system.RecipientKindTopic, Subject: topicCharacter}, model.CreateCharacterRequest{Name: "Alice"}); err != nil {
	// 	slog.Error("Failed to publish create character message", "error", err)
	// 	return
	// }

	handlerTicker, err := sys.Spawn(ctx, "TickerActor")
	if err != nil {
		slog.Error("Failed to spawn actor", "error", err)
		return
	}

	slog.Info("Actors spawned successfully", "tickerActorID", handlerTicker.GetActor().GetID(), "characterStoreID", characterStore.GetActor().GetID())

	// Wait for interrupt signal (Ctrl+C)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	slog.Info("Interrupt received, stopping actor")
	handlerTicker.Stop()
	handlerTicker.WaitForTermination()

	time.Sleep(2 * time.Second) // Wait for ticker to stop before stopping character store

	characterStore.Stop()
	characterStore.WaitForTermination()

	server.Shutdown(ctx)
}
