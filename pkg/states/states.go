//
// Copyright (c) 2018 Intel Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// SPDX-License-Identifier: EPL-2.0
//

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
