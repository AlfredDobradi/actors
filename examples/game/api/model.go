package api

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
	CharacterName string         `json:"character_name"`
	Action        string         `json:"action"`
	Context       map[string]any `json:"context"`
}

type StopActionRequest struct {
	CharacterName string `json:"character_name"`
}
