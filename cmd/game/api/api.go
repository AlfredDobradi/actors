package api

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/alfreddobradi/actors/cmd/game/api/handler"
	"github.com/alfreddobradi/actors/cmd/game/api/middleware"
	"github.com/alfreddobradi/actors/cmd/game/api/state"
	"github.com/alfreddobradi/actors/pkg/config"
	"github.com/alfreddobradi/actors/pkg/database"
	"github.com/alfreddobradi/actors/pkg/system"
	"github.com/gorilla/mux"
)

type Server struct {
	*http.Server

	context  *state.Context
	listener net.Listener
}

func NewServer(sys *system.System, kv database.KeyValue, db database.Store) *Server {
	router := mux.NewRouter()

	listener, err := net.Listen("tcp", config.GetConfig().Addr)
	if err != nil {
		slog.Error("Failed to start listener", "error", err)
		return nil
	}

	stateCtx := state.New(kv, db, sys)

	s := &Server{
		Server: &http.Server{
			Handler:      router,
			ReadTimeout:  60 * time.Second,
			WriteTimeout: 60 * time.Second,
			IdleTimeout:  120 * time.Second,
		},
		context:  &stateCtx,
		listener: listener,
	}

	router.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slog.Warn("Received request for unknown route", "method", r.Method, "path", r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
		if _, err := w.Write([]byte("Not found")); err != nil {
			slog.Warn("Failed to write response", "error", err)
		}
	})

	router.HandleFunc("/auth/account", handler.HandleCreateAccount(s.context)).Methods(http.MethodPost)
	router.HandleFunc("/auth/session", handler.HandleCreateSession(s.context)).Methods(http.MethodPost)
	router.HandleFunc("/auth/session", handler.HandleDeleteSession(s.context)).Methods(http.MethodDelete)

	admin := router.PathPrefix("/admin").Subrouter()
	// admin.Use(middleware.Authorization(db))
	admin.HandleFunc("/accounts", handler.HandleAdminGetAccounts(s.context)).Methods(http.MethodGet)
	admin.HandleFunc("/accounts/{accountId}", handler.HandleAdminGetAccount(s.context)).Methods(http.MethodGet)
	admin.HandleFunc("/config", handler.HandleAdminForceRefreshConfig(s.context)).Methods(http.MethodGet)

	a := router.PathPrefix("/account").Subrouter()
	a.Use(middleware.Authorization(db))
	a.HandleFunc("/tavern", handler.HandleCreateTavern(s.context)).Methods(http.MethodPost)
	a.HandleFunc("/tavern/hire", handler.HandleHireCharacter(s.context)).Methods(http.MethodPost)
	a.HandleFunc("/tavern/characters", handler.HandleGetCharacters(s.context)).Methods(http.MethodGet)
	a.HandleFunc("/tavern/characters/{characterId}", handler.HandleGetCharacter(s.context)).Methods(http.MethodGet)

	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("Welcome to the game API")); err != nil {
			slog.Warn("Failed to write response", "error", err)
		}
	})

	return s
}

func (s *Server) Shutdown(ctx context.Context) error {
	slog.Info("Shutting down API server")
	return s.Server.Shutdown(ctx)
}

func (s *Server) Addr() string {
	return s.listener.Addr().String()
}

func (s *Server) Start() error {
	slog.Info("Starting API server", "addr", s.Addr())
	return s.Serve(s.listener)
}

func (s *Server) Stop(ctx context.Context) error {
	return s.Shutdown(ctx)
}
