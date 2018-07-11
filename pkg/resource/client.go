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

package resource

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// Client manipulates Kubernetes API resources backed by template files.
type Client interface {
	// Reify returns the raw request body given the supplied template values.
	Reify(templateValues interface{}) ([]byte, error)
	// Create creates a new object using the supplied data object for
	// template expansion.
	Create(namespace string, templateValues interface{}) error
	// Delete deletes the object.
	Delete(namespace string, name string) error
	// Get retrieves the object.
	Get(namespace, name string) (runtime.Object, error)
	// List lists objects based on group, version and kind.
	List(namespace string, labels map[string]string) ([]metav1.Object, error)
	// Update updates the object
	Update(object runtime.Object) (runtime.Object, error)
	// Plural returns the plural form of the resource.
	Plural() string
}
