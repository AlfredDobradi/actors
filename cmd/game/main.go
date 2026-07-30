package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/alfreddobradi/actors/cmd/game/actor"
	"github.com/alfreddobradi/actors/cmd/game/api"
	"github.com/alfreddobradi/actors/cmd/game/logging"
	"github.com/alfreddobradi/actors/pkg/config"
	"github.com/alfreddobradi/actors/pkg/database"
	"github.com/alfreddobradi/actors/pkg/database/kv/etcd"
	"github.com/alfreddobradi/actors/pkg/database/store/postgres"
	"github.com/alfreddobradi/actors/pkg/system"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		slog.Warn("No .env file found, relying on environment variables")
	}

	if err := config.Load("./config.yaml"); err != nil {
		slog.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	logging.Init()

	registry := system.NewRegistry()
	actor.InitFactories(registry)

	cfg := config.GetConfig()

	var (
		kv    database.KeyValue
		kvErr error
	)
	if kv, kvErr = etcd.New(cfg.KV.Hosts); kvErr != nil {
		slog.Error("Failed to initialize key-value store", "error", kvErr)
		os.Exit(1)
	}

	var (
		db    *postgres.Connection
		dbErr error
	)
	if db, dbErr = postgres.New(cfg.Database.DSN()); dbErr != nil {
		slog.Error("Failed to initialize database", "error", dbErr)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sys, err := system.NewSystem(registry, db, kv)
	if err != nil {
		slog.Error("Failed to create system", "error", err)
		os.Exit(1)
	}

	apiServer := api.NewServer(sys, kv, db)

	go apiServer.Start() //nolint

	handlerTicker, err := sys.Spawn(ctx, "TickerActor")
	if err != nil {
		slog.Error("Failed to spawn actor", "error", err)
		return
	}

	slog.Info("Actors spawned successfully", "ticker_actor_id", handlerTicker.GetActor().GetID())

	// Wait for interrupt signal (Ctrl+C)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	logging.LogError(apiServer.Shutdown(ctx), "Failed to shutdown API server")
	logging.LogError(sys.Shutdown(ctx), "Failed to shutdown system")
	logging.LogError(kv.Close(ctx), "Failed to close database")
}
