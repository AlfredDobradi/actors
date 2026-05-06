package api

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/alfreddobradi/actors/examples/game/api/middleware"
	"github.com/alfreddobradi/actors/examples/game/database"
	"github.com/alfreddobradi/actors/pkg/system"
	"github.com/gorilla/mux"
)

type Server struct {
	*http.Server

	sys *system.System
	db  database.DB
}

func NewServer(addr string, sys *system.System, db database.DB) *Server {
	router := mux.NewRouter()

	s := &Server{
		Server: &http.Server{
			Addr:    addr,
			Handler: router,
		},
		sys: sys,
		db:  db,
	}

	router.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slog.Info("not found", "method", r.Method, "path", r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Not found"))
	})

	router.HandleFunc("/auth/account", s.handleCreateAccount).Methods(http.MethodPost)
	router.HandleFunc("/auth/session", s.handleCreateSession).Methods(http.MethodPost)
	router.HandleFunc("/auth/session", s.handleDeleteSession).Methods(http.MethodDelete)

	c := router.PathPrefix("/character").Subrouter()
	c.Use(middleware.Authorization(db))
	c.HandleFunc("/", s.handleGetCharacter).Methods(http.MethodPost)
	c.HandleFunc("/action", s.handleStartAction).Methods(http.MethodPost)
	c.HandleFunc("/action", s.handleStopAction).Methods(http.MethodDelete)

	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Welcome to the game API"))
	})

	return s
}

func (s *Server) Start() error {
	return s.ListenAndServe()
}

func (s *Server) Stop(ctx context.Context) error {
	return s.Shutdown(ctx)
}
