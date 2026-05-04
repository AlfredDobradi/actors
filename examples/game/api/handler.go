package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/alfreddobradi/actors/examples/game/actor"
	"github.com/alfreddobradi/actors/examples/game/game"
	"github.com/alfreddobradi/actors/pkg/system"
	"github.com/davecgh/go-spew/spew"
	"github.com/google/uuid"
)

func (s *Server) handleGetCharacter(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var httpReq GetCharacterRequest
	if err := decoder.Decode(&httpReq); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	req := actor.GetCharacterRequest{Name: httpReq.Name}

	ctx := context.WithValue(r.Context(), system.ContextKeySpanID, uuid.New())

	characterData, err := s.sys.Request(ctx, uuid.Nil, system.Recipient{Kind: system.RecipientKindTopic, Subject: "character"}, req)
	if err != nil {
		http.Error(w, "Failed to request character", http.StatusInternalServerError)
		return
	}

	spew.Fdump(w, characterData)
}

func (s *Server) handleStartAction(w http.ResponseWriter, r *http.Request) {
	ctx := context.WithValue(r.Context(), system.ContextKeySpanID, uuid.New())
	decoder := json.NewDecoder(r.Body)
	var httpReq StartActionRequest
	if err := decoder.Decode(&httpReq); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	characterReq := actor.GetCharacterRequest{Name: httpReq.CharacterName}

	characterData, err := s.sys.Request(ctx, uuid.Nil, system.Recipient{Kind: system.RecipientKindTopic, Subject: "character"}, characterReq)
	if err != nil {
		http.Error(w, "Failed to request character", http.StatusInternalServerError)
		return
	}

	var action game.Action
	switch httpReq.Action {
	case "gather":
		if resourceName, ok := httpReq.Context["resource"].(string); ok {
			resource, found := game.ResourceByName(resourceName)
			if !found {
				http.Error(w, "Resource not found", http.StatusBadRequest)
				return
			}
			action = &game.GatherAction{Resource: resource}
		} else {
			http.Error(w, "Invalid resource", http.StatusBadRequest)
			return
		}
	case "fight":
		action = &game.FightAction{}
	default:
		http.Error(w, "Invalid action type", http.StatusBadRequest)
		return
	}

	startReq := actor.StartActionRequest{
		CharacterID: characterData.(*game.Character).ID,
		Action:      action,
	}

	if err := s.sys.Publish(ctx, uuid.Nil, system.Recipient{Kind: system.RecipientKindTopic, Subject: "character"}, startReq); err != nil {
		http.Error(w, "Failed to request character action start", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Action " + httpReq.Action + " started successfully"))
}

func (s *Server) handleStopAction(w http.ResponseWriter, r *http.Request) {
	ctx := context.WithValue(r.Context(), system.ContextKeySpanID, uuid.New())
	decoder := json.NewDecoder(r.Body)
	var httpReq StopActionRequest
	if err := decoder.Decode(&httpReq); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	characterReq := actor.GetCharacterRequest{Name: httpReq.CharacterName}

	characterData, err := s.sys.Request(ctx, uuid.Nil, system.Recipient{Kind: system.RecipientKindTopic, Subject: "character"}, characterReq)
	if err != nil {
		http.Error(w, "Failed to request character", http.StatusInternalServerError)
		return
	}

	stopReq := actor.StopActionRequest{
		CharacterID: characterData.(*game.Character).ID,
	}

	if err := s.sys.Publish(ctx, uuid.Nil, system.Recipient{Kind: system.RecipientKindTopic, Subject: "character"}, stopReq); err != nil {
		http.Error(w, "Failed to request character action stop", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Action stopped successfully"))
}
