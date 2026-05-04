package api

import (
	"context"
	"net/http"

	"github.com/alfreddobradi/actors/pkg/system"
	"github.com/gorilla/mux"
)

type Server struct {
	*http.Server

	sys *system.System
}

func NewServer(addr string, sys *system.System) *Server {
	router := mux.NewRouter()

	s := &Server{
		Server: &http.Server{
			Addr:    addr,
			Handler: router,
		},
		sys: sys,
	}

	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Welcome to the game API"))
	})
	router.HandleFunc("/character", s.handleGetCharacter).Methods(http.MethodPost)
	router.HandleFunc("/character/action", s.handleStartAction).Methods(http.MethodPost)
	router.HandleFunc("/character/action", s.handleStopAction).Methods(http.MethodDelete)

	return s
}

func (s *Server) Start() error {
	return s.ListenAndServe()
}

func (s *Server) Stop(ctx context.Context) error {
	return s.Shutdown(ctx)
}
