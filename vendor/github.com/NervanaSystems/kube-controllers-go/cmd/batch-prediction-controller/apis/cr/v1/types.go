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

	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/NervanaSystems/kube-controllers-go/pkg/states"
	"k8s.io/apimachinery/pkg/runtime"
)

const GroupName = "aipg.intel.com"

const Version = "v1"

// The kind of the crd
const BatchPredictionResourceKind = "BatchPrediction"

// The singular form of the crd
const BatchPredictionResourceSingular = "batchprediction"

// The plural form of the crd
const BatchPredictionResourcePlural = "batchpredictions"

var (
	// GVK unambiguously identifies the batch prediction kind.
	GVK = schema.GroupVersionKind{
		Group:   GroupName,
		Version: Version,
		Kind:    BatchPredictionResourceKind,
	}
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type BatchPrediction struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              BatchPredictionSpec   `json:"spec"`
	Status            BatchPredictionStatus `json:"status,omitempty"`
}

func (s *BatchPrediction) Name() string {
	return s.ObjectMeta.Name
}

func (s *BatchPrediction) Namespace() string {
	return s.ObjectMeta.Namespace
}

func (s *BatchPrediction) JSON() (string, error) {
	data, err := json.Marshal(s)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func (s *BatchPrediction) GetStatusState() states.State {
	return s.Status.State
}

func (s *BatchPrediction) GetSpecState() states.State {
	return s.Spec.State
}

func (s *BatchPrediction) SetStatusStateWithMessage(state states.State, msg string) {
	s.Status.State = state
	s.Status.Message = msg
}

// BatchPredictionSpec is the spec for the crd.
type BatchPredictionSpec struct {
	JobID        string       `json:"jobID"`
	State        states.State `json:"state"`
	ResourceSpec ResourceSpec `json:"resourceSpec"`
}

// ResourceSpec defines the compute resources required.
type ResourceSpec struct {
	Requests map[string]resource.Quantity `json:"requests"`
}

// BatchPredictionStatus is the status for the crd.
type BatchPredictionStatus struct {
	State   states.State `json:"state,omitempty"`
	Message string       `json:"message,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type BatchPredictionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []BatchPrediction `json:"items"`
}

// GetItems returns the list of items to be used in the List api call for crs
func (mtl *BatchPredictionList) GetItems() []runtime.Object {
	var result []runtime.Object
	for _, item := range mtl.Items {
		mtCopy := item
		result = append(result, &mtCopy)
	}
	return result
}
