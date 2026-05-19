package actor

import "fmt"

// TODO: organizer errors better

type ErrInvalidMessage struct {
	Kind string
}

func (ErrInvalidMessage) IsRecoverable() bool {
	return false
}

func (e ErrInvalidMessage) Error() string {
	return fmt.Sprintf("invalid message of kind %s", e.Kind)
}

func NewErrInvalidMessage(kind string) ErrInvalidMessage {
	return ErrInvalidMessage{Kind: kind}
}

type ErrTavernRequired struct{}

func (ErrTavernRequired) IsRecoverable() bool {
	return false
}

func (ErrTavernRequired) Error() string {
	return "tavern required for that action"
}

type ErrTavernExists struct{}

func (ErrTavernExists) IsRecoverable() bool {
	return false
}

func (ErrTavernExists) Error() string {
	return "tavern already exists"
}

type ErrResponseFailed struct {
	Err error
}

func (e ErrResponseFailed) IsRecoverable() bool {
	return true
}

func (e ErrResponseFailed) Error() string {
	return fmt.Sprintf("failed to send response: %v", e.Err)
}

func NewErrResponseFailed(err error) ErrResponseFailed {
	return ErrResponseFailed{Err: err}
}
