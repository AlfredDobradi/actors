package game

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/google/uuid"
)

type Tavern struct {
	mx *sync.Mutex

	characters map[uuid.UUID]*Character
}

func NewTavern() *Tavern {
	return &Tavern{
		mx: &sync.Mutex{},

		characters: make(map[uuid.UUID]*Character),
	}
}

func (t *Tavern) UnmarshalJSON(data []byte) error {
	var aux struct {
		Characters map[uuid.UUID]Character `json:"characters"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	if t.mx == nil {
		t.mx = &sync.Mutex{}
	}

	t.mx.Lock()
	defer t.mx.Unlock()

	t.characters = make(map[uuid.UUID]*Character)
	for id, character := range aux.Characters {
		t.characters[id] = &character
	}
	return nil
}

func (t *Tavern) MarshalJSON() ([]byte, error) {
	t.mx.Lock()
	defer t.mx.Unlock()

	characters := make(map[uuid.UUID]Character)
	for id, character := range t.characters {
		characters[id] = *character
	}

	aux := struct {
		Characters map[uuid.UUID]Character `json:"characters"`
	}{
		Characters: characters,
	}
	return json.Marshal(aux)
}

func (t *Tavern) AddCharacter(character *Character) {
	t.mx.Lock()
	defer t.mx.Unlock()
	t.characters[character.ID] = character
}

func (t *Tavern) Characters() map[uuid.UUID]*Character {
	t.mx.Lock()
	defer t.mx.Unlock()
	return t.characters
}

func (t *Tavern) GetCharacter(characterID uuid.UUID) (*Character, bool) {
	t.mx.Lock()
	defer t.mx.Unlock()
	if character, exists := t.characters[characterID]; exists {
		return character, true
	}
	return nil, false
}

func (t *Tavern) ProcessTick(ctx context.Context) {
	t.mx.Lock()
	defer t.mx.Unlock()

	wg := sync.WaitGroup{}
	for id, character := range t.characters {
		wg.Add(1)
		go func(id uuid.UUID, character *Character) {
			defer wg.Done()
			character.ProcessTick(ctx)
			t.characters[id] = character
		}(id, character)
	}
	wg.Wait()
}
