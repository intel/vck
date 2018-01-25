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
const StreamPredictionResourceKind = "StreamPrediction"

// The singular form of the crd
const StreamPredictionResourceSingular = "streamprediction"

// The plural form of the crd
const StreamPredictionResourcePlural = "streampredictions"

var (
	// GVK unambiguously identifies the stream predicition kind.
	GVK = schema.GroupVersionKind{
		Group:   GroupName,
		Version: Version,
		Kind:    StreamPredictionResourceKind,
	}
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type StreamPrediction struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              StreamPredictionSpec   `json:"spec"`
	Status            StreamPredictionStatus `json:"status,omitempty"`
}

func (s *StreamPrediction) Name() string {
	return s.ObjectMeta.Name
}

func (s *StreamPrediction) Namespace() string {
	return s.ObjectMeta.Namespace
}

func (s *StreamPrediction) JSON() (string, error) {
	data, err := json.Marshal(s)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func (s *StreamPrediction) GetStatusState() states.State {
	return s.Status.State
}

func (s *StreamPrediction) GetSpecState() states.State {
	return s.Spec.State
}

func (s *StreamPrediction) SetStatusStateWithMessage(state states.State, msg string) {
	s.Status.State = state
	s.Status.Message = msg
}

// StreamPredictionState is the current job state.
type StreamPredictionState string

// StreamPredictionSpec is the spec for the crd.
type StreamPredictionSpec struct {
	NeonRepoSpec    NeonRepoSpec    `json:"neonRepoSpec"`
	SecuritySpec    SecuritySpec    `json:"securitySpec"`
	StreamDataSpec  StreamDataSpec  `json:"streamDataSpec"`
	KryptonRepoSpec KryptonRepoSpec `json:"kryptonRepoSpec"`
	ResourceSpec    ResourceSpec    `json:"resourceSpec"`
	State           states.State    `json:"state"`
}

type KryptonRepoSpec struct {
	RepoURL      string `json:"repoURL"`
	Commit       string `json:"commit"`
	Image        string `json:"image"`
	SidecarImage string `json:"sidecarImage"`
}

type NeonRepoSpec struct {
	RepoURL string `json:"repoURL"`
	Commit  string `json:"commit"`
}

type StreamDataSpec struct {
	ModelPRM         string `json:"modelPRM"`
	ModelPath        string `json:"modelPath"`
	DatasetPath      string `json:"datasetPath"`
	ExtraFilename    string `json:"extraFilename"`
	CustomCodeURL    string `json:"customCodeURL"`
	CustomCommit     string `json:"customCommit"`
	AWSPath          string `json:"awsPath"`
	AWSDefaultRegion string `json:"awsDefaultRegion"`
	StreamID         int    `json:"streamID"`
	StreamName       string `json:"streamName"`
}

type SecuritySpec struct {
	PresignedToken string `json:"presignedToken"`
	JWTToken       string `json:"jwtToken"`
}

type ResourceSpec struct {
	Requests map[string]resource.Quantity `json:"requests"`
}

// StreamPredictionStatus is the status for the crd.
type StreamPredictionStatus struct {
	State   states.State `json:"state,omitempty"`
	Message string       `json:"message,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type StreamPredictionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []StreamPrediction `json:"items"`
}

// GetItems returns the list of items to be used in the List api call for crs
func (spl *StreamPredictionList) GetItems() []runtime.Object {
	var result []runtime.Object
	for _, item := range spl.Items {
		spCopy := item
		result = append(result, &spCopy)
	}
	return result
}
