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

package v1

import (
	"encoding/json"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/kubeflow/experimental-kvc/pkg/states"
)

const (
	GroupName string = "kvc.kubeflow.org"

	Version string = "v1"

	// The kind of the crd.
	VolumeManagerResourceKind string = "VolumeManager"

	// The singular form of the crd.
	VolumeManagerResourceSingular string = "volumemanager"

	// The plural form of the crd.
	VolumeManagerResourcePlural string = "volumemanagers"

	// The message for a successful volumemanager status.
	SuccessfulVolumeStatusMessage string = "success"
)

var (
	// GVK unambiguously identifies the volume manager kind.
	GVK = schema.GroupVersionKind{
		Group:   GroupName,
		Version: Version,
		Kind:    VolumeManagerResourceKind,
	}
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Volumemanager is the spec for a VolumeManager CR
type VolumeManager struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              VolumeManagerSpec   `json:"spec"`
	Status            VolumeManagerStatus `json:"status,omitempty"`
}

func (s *VolumeManager) JSON() (string, error) {
	data, err := json.Marshal(s)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (s *VolumeManager) GetStatusState() states.State {
	return s.Status.State
}

func (s *VolumeManager) GetSpecState() states.State {

	return s.Spec.State
}

func (s *VolumeManager) SetStatusStateWithMessage(state states.State, msg string) {
	s.Status.State = state
	s.Status.Message = msg
}

// DataSourceType is the type of the data source (e.g., S3, NFS).
type DataSourceType string

// VolumeConfig contains all the configuration required for a volume.
type VolumeConfig struct {
	ID           string              `json:"id"`
	Replicas     int                 `json:"replicas"`
	SourceType   DataSourceType      `json:"sourceType"`
	EndpointURL  string              `json:"endpointURL"`
	SourceURL    string              `json:"sourceURL"`
	AccessMode   string              `json:"accessMode"`
	Capacity     string              `json:"capacity"`
	NodeAffinity corev1.NodeAffinity `json:"nodeAffinity"`
	Tolerations  []corev1.Toleration `json:"tolerations"`
	Labels       map[string]string   `json:"labels"`
	Options      map[string]string   `json:"options"`
}

// VolumeManagerSpec is the spec for the crd.
type VolumeManagerSpec struct {
	VolumeConfigs []VolumeConfig `json:"volumeConfigs"`
	State         states.State   `json:"state"`
}

// Volume provides the details on volume source and node affinity.
type Volume struct {
	ID           string              `json:"id"`
	VolumeSource corev1.VolumeSource `json:"volumeSource"`
	NodeAffinity corev1.NodeAffinity `json:"nodeAffinity"`
	Message      string              `json:"message,omitempty"`
}

// VolumeManagerStatus is the status for the crd.
type VolumeManagerStatus struct {
	Volumes []Volume     `json:"volumes"`
	State   states.State `json:"state,omitempty"`
	Message string       `json:"message,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// VolumemanagerList is the list of VolumeManager resources
type VolumeManagerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []VolumeManager `json:"items"`
}

// GetItems returns the list of items to be used in the List api call for crs
func (vml *VolumeManagerList) GetItems() []runtime.Object {
	var result []runtime.Object
	for _, item := range vml.Items {
		vmCopy := item
		result = append(result, &vmCopy)
	}
	return result
}
