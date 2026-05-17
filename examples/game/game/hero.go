package game

import (
	"context"
	"sync"

	"github.com/google/uuid"
)

type Tavern struct {
	mx *sync.Mutex

	characters map[uuid.UUID]*Character
}

func NewTavern() *Tavern {
	return &Tavern{
		mx:         &sync.Mutex{},
		characters: make(map[uuid.UUID]*Character),
	}
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
