package model

import (
	"github.com/alfreddobradi/actors/examples/game/game"
	"github.com/google/uuid"
)

type StartActionMessage struct {
	CharacterID uuid.UUID      `json:"character_id"`
	Action      game.Action    `json:"action"`
	Context     map[string]any `json:"context"`
}

type StopActionMessage struct {
	CharacterID uuid.UUID `json:"character_id"`
}
