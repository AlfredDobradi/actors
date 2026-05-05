package model

import (
	"github.com/alfreddobradi/actors/examples/game/game"
	"github.com/google/uuid"
)

type CreateCharacterRequest struct {
	Name string
}

type GetCharacterRequest struct {
	Name string
}

type StartActionRequest struct {
	CharacterID uuid.UUID
	Action      game.Action
}

type StopActionRequest struct {
	CharacterID uuid.UUID
}
