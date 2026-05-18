package actor

import "fmt"

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
