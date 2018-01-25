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
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/NervanaSystems/kube-controllers-go/pkg/states"
)

const GroupName = "aipg.intel.com"

const Version = "v1"

// The kind of the crd
const VolumeManagerResourceKind = "VolumeManager"

// The singular form of the crd
const VolumeManagerResourceSingular = "volumemanager"

// The plural form of the crd
const VolumeManagerResourcePlural = "volumemanagers"

var (
	// GVK unambiguously identifies the volume manager kind.
	GVK = schema.GroupVersionKind{
		Group:   GroupName,
		Version: Version,
		Kind:    VolumeManagerResourceKind,
	}
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type VolumeManager struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              VolumeManagerSpec   `json:"spec"`
	Status            VolumeManagerStatus `json:"status,omitempty"`
}

func (s *VolumeManager) Name() string {
	return s.ObjectMeta.Name
}

func (s *VolumeManager) Namespace() string {
	return s.ObjectMeta.Namespace
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

// VolumeManagerSpec is the spec for the crd.
type VolumeManagerSpec struct {
	Volumes []VolumeConfig `json:"volumes"`
	State   states.State   `json:"state"`
}

// VolumeConfig contains all the configuration required for the volumes.
type VolumeConfig struct {
	ID         string            `json:"id"`
	Replicas   string            `json:"replicas"`
	SourceType DataSourceType    `json:"sourceType"`
	SourceURL  string            `json:"sourceURL"`
	MountPath  string            `json:"mountPath"`
	AccessMode string            `json:"accessMode"`
	Options    map[string]string `json:"options"`
}

// DataSourceType is the type of the data source (e.g., S3, NFS).
type DataSourceType string

// VolumeManagerStatus is the status for the crd.
type VolumeManagerStatus struct {
	VolumeClaims []VolumeClaimStatus `json:"volumeClaims"`
	State        states.State        `json:"state,omitempty"`
	Message      string              `json:"message,omitempty"`
}

// VolumeClaimStatus provides the details on PVC to claim for corresponding
// volumes.
type VolumeClaimStatus struct {
	ID      string `json:"id"`
	PVCName string `json:"pvcName"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
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
