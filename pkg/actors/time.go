package actors

import "github.com/google/uuid"

type TimeActor struct {
	ID uuid.UUID
}

func (t *TimeActor) GetID() uuid.UUID {
	return t.ID
}

func (t *TimeActor) GetKind() string {
	return "TimeActor"
}
