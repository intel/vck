/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import (
	"encoding/json"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/NervanaSystems/kube-controllers-go/pkg/states"
	"k8s.io/apimachinery/pkg/runtime"
)

// GroupName is the group name used in this package.
const GroupName = "cr.client-go.k8s.io"

const Version = "v1"

const ExampleResourceKind = "Example"

const ExampleResourceSingular = "example"

const ExampleResourcePlural = "examples"

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type Example struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              ExampleSpec   `json:"spec"`
	Status            ExampleStatus `json:"status,omitempty"`
}

func (e *Example) Name() string {
	return e.ObjectMeta.Name
}

func (e *Example) Namespace() string {
	return e.ObjectMeta.Namespace
}

func (e *Example) JSON() (string, error) {
	data, err := json.Marshal(e)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func (e *Example) GetStatusState() states.State {
	return e.Status.State
}

func (e *Example) GetSpecState() states.State {
	return e.Spec.State
}

func (e *Example) SetStatusStateWithMessage(state states.State, msg string) {
	e.Status.State = state
	e.Status.Message = msg
}

type ExampleSpec struct {
	State states.State `json:"state"`
	Foo   string       `json:"foo"`
	Bar   bool         `json:"bar"`
}

type ExampleStatus struct {
	State   states.State `json:"state"`
	Message string       `json:"message,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ExampleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []Example `json:"items"`
}

// GetItems returns the list of items to be used in the List api call for crs
func (el *ExampleList) GetItems() []runtime.Object {
	var result []runtime.Object
	for _, item := range el.Items {
		ecCopy := item
		result = append(result, &ecCopy)
	}
	return result
}
