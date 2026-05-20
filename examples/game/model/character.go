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

/*
type Character struct {
	ID         uuid.UUID `json:"id"`
	Name       string    `json:"name"`
	Level      int       `json:"level"`
	Experience int       `json:"experience"`
	Status     uint8     `json:"status"`
	Cooldown   int       `json:"cooldown"`
	Action     Action    `json:"-"`

	Inventory *Inventory `json:"inventory"`
}
*/
type CharacterDetails struct {
	ID         uuid.UUID `json:"id"`
	Name       string    `json:"name"`
	Level      int       `json:"level"`
	Experience int       `json:"experience"`
	Progress   int       `json:"progress"`
	Action     string    `json:"action"`
}

func DetailsFromCharacter(c *game.Character) CharacterDetails {
	action := "is not currently doing anything"
	if c.Action != nil {
		action = c.Action.String()
	}

	return CharacterDetails{
		ID:         c.ID,
		Name:       c.Name,
		Level:      c.Level,
		Experience: c.Experience,
		Progress:   c.Cooldown,
		Action:     action,
	}
}
