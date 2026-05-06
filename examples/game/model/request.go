package model

type CreateCharacterRequest struct {
	Name string `json:"name"`
}

type GetCharacterRequest struct {
	Name string `json:"name"`
}

type GetCharacterResponse struct {
	Name       string         `json:"name"`
	Level      int            `json:"level"`
	Experience int            `json:"experience"`
	Inventory  map[string]int `json:"inventory"`
}

type StartActionRequest struct {
	CharacterID string         `json:"character_id"`
	Action      string         `json:"action"`
	Context     map[string]any `json:"context"`
}

type StopActionRequest struct {
	CharacterID string `json:"character_id"`
}
