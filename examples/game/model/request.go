package model

type CreateCharacterRequest struct {
	Name string `json:"name"`
}

type GetCharacterRequest struct {
	ID string `json:"id"`
}

type GetCharactersRequest struct{}

type GetCharacterResponse struct {
	Status  string           `json:"status"`
	Details CharacterDetails `json:"details,omitempty"`
	Error   string           `json:"error,omitempty"`
}

type GetCharactersResponse struct {
	Status  string             `json:"status"`
	Details []CharacterDetails `json:"details,omitempty"`
	Error   string             `json:"error,omitempty"`
}

type StartActionRequest struct {
	CharacterID string         `json:"character_id"`
	Action      string         `json:"action"`
	Context     map[string]any `json:"context"`
}

type StopActionRequest struct {
	CharacterID string `json:"character_id"`
}

type NewTavernRequest struct{}

type NewTavernResponse struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

type HireCharacterRequest struct{}

type HireCharacterResponse struct {
	CharacterID   string `json:"character_id,omitempty"`
	CharacterName string `json:"character_name,omitempty"`
	OK            bool   `json:"ok"`
	Error         string `json:"error,omitempty"`
}
