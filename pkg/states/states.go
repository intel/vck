package states

// The State type's inhabitants comprise a VolumeManager's state space.
type State string

const (
	// Pending - In this state, the VolumeManager CR has been created, but its sub-resources are pending.
	Pending State = "Pending"

	// Running - This is the _ready_ state for a VolumeManager CR.
	// In this state, it is running as expected.
	Running State = "Running"

	// Completed - A `Completed` a VolumeManager CR has been undeployed. `Completed` is a terminal state.
	Completed State = "Completed"

	// Failed - VolumeManager CR is in a `Failed` state if an error has caused it to no longer be running as expected.
	Failed State = "Failed"
)
