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
	"github.com/alfreddobradi/actors/examples/game/config"
	"github.com/alfreddobradi/actors/examples/game/database"
	"github.com/alfreddobradi/actors/examples/game/database/etcd"
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

	if err := config.Load("./config.yaml"); err != nil {
		slog.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	registry := system.NewRegistry()
	actor.InitFactories(registry)

	var (
		db    database.DB
		dbErr error
	)
	if db, dbErr = etcd.New([]string{"localhost:22379", "localhost:22479", "localhost:22579"}); dbErr != nil {
		slog.Error("Failed to initialize database", "error", dbErr)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sys, err := system.NewSystem(registry, db)
	if err != nil {
		slog.Error("Failed to create system", "error", err)
		os.Exit(1)
	}

	apiServer := api.NewServer(sys, db)
	go apiServer.Start()

	characterStore, err := sys.Spawn(ctx, "CharacterStore",
		system.WithSubscription(topicTicks),
		system.WithSubscription(topicCharacter),
	)
	if err != nil {
		slog.Error("Failed to spawn actor", "error", err)
		return
	}

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

	apiServer.Shutdown(ctx)
	sys.Shutdown(ctx)
	db.Close(ctx)
}
