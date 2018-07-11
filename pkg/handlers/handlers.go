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

package handlers

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	vckv1 "github.com/IntelAI/vck/pkg/apis/vck/v1"
)

// DataHandler is the interface which defines the handler methods
type DataHandler interface {
	GetSourceType() vckv1.DataSourceType
	OnAdd(namespace string, vc vckv1.VolumeConfig, controllerRef metav1.OwnerReference) vckv1.Volume
	OnDelete(namespace string, vc vckv1.VolumeConfig, vStatus vckv1.Volume, controllerRef metav1.OwnerReference)
}

const (
	vckNamePrefix string = "vck-resource-"
)
