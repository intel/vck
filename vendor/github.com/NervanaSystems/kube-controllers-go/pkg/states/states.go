package states

import (
	"errors"
)

// ErrInvalidState is the error returned when a given state does not exist
// in the state space for any job type.
var ErrInvalidState = errors.New("invalid state given")

// The State type's inhabitants comprise a job's state space.
type State string

const (
	// Pending In this state, a job has been created, but its sub-resources are pending.
	Pending State = "Pending"

	// Running This is the _ready_ state for a job.
	// In this state, it is running as expected.
	Running State = "Running"

	// Completed A `Completed` job has been undeployed. `Completed` is a terminal state.
	Completed State = "Completed"

	// Failed A job is in an `Failed` state if an error has caused it to no longer be running as expected.
	Failed State = "Failed"
)

// IsTerminal returns true if the provided state is terminal.
func IsTerminal(state State) bool {
	return (state == Completed || state == Failed)
}

// IsOneOf returns true if this state is in the supplied list.
func (s State) IsOneOf(targets ...State) bool {
	for _, target := range targets {
		if s == target {
			return true
		}
	}
	return false
}
