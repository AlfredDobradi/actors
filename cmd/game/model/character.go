package model

import (
	"github.com/alfreddobradi/actors/cmd/game/game"
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

type CharacterDetails struct {
	ID         uuid.UUID `json:"id"`
	Name       string    `json:"name"`
	Level      int       `json:"level"`
	Experience int       `json:"experience"`
	Health     int       `json:"health"`
	Energy     int       `json:"energy"`
	Gold       int       `json:"gold"`
	Progress   int       `json:"progress"`
	Action     string    `json:"action"`
}

func DetailsFromCharacter(c *game.Hero) CharacterDetails {
	action := "is not currently doing anything"
	if c.Action != nil {
		action = c.Action.String()
	}

	return CharacterDetails{
		ID:         c.ID,
		Name:       c.Name,
		Level:      c.Level,
		Experience: c.Experience,
		Health:     c.Health,
		Energy:     c.Energy,
		Gold:       c.Gold,
		Progress:   c.Cooldown,
		Action:     action,
	}
}
