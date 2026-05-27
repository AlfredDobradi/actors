package actor

import (
	"fmt"
	"time"
)

// TODO: organize errors better

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

type AccountError struct {
	Err error
}

func (e AccountError) IsRecoverable() bool {
	return false
}

func (e AccountError) Error() string {
	return fmt.Sprintf("account error: %v", e.Err)
}

func NewAccountError(err error) AccountError {
	return AccountError{Err: err}
}

type ErrReplayFailed struct {
	Err            error
	Timestamp      time.Time
	TicksDone      int
	TicksRemaining int
}

func (e ErrReplayFailed) IsRecoverable() bool {
	return true
}

func (e ErrReplayFailed) Error() string {
	return fmt.Sprintf("replay failed: %v (ticks done: %d, ticks remaining: %d, new timestamp: %v)", e.Err, e.TicksDone, e.TicksRemaining, e.Timestamp)
}
