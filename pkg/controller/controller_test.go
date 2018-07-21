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

package controller

import (
	"context"
	"testing"

	vckv1alpha1 "github.com/ppkube/vck/pkg/apis/vck/v1alpha1"
	"github.com/ppkube/vck/pkg/client/clientset/versioned/fake"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type FakeHook struct {
	addCalled    bool
	deleteCalled bool
	updateCalled bool
	counter      chan string
}

func (f *FakeHook) Add(obj interface{}) {
	f.counter <- "Add"
	f.addCalled = true
}

func (f *FakeHook) Update(oldObj, newObj interface{}) {
	f.counter <- "Update"
	f.updateCalled = true
}

func (f *FakeHook) Delete(obj interface{}) {
	f.counter <- "Delete"
	f.deleteCalled = true
}

func TestController(t *testing.T) {

	// TODO: Add update and delete tests. They are for some reason not getting called.

	// Create a counter to track the calls
	counter := make(chan string, 3)

	// Fake hook to verify the calls
	hook := FakeHook{counter: counter}

	// Get a context
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	// Create a fake client to pass in to the informers
	namespace := "test"
	fakeClient := fake.NewSimpleClientset()

	volumeManagerClient := fakeClient.Vck().VolumeManagers(namespace)
	controller := New(&hook, fakeClient)

	// Start the controller
	go controller.Run(ctx, namespace)

	// Create the CR using the fake client
	volume, err := volumeManagerClient.Create(&vckv1alpha1.VolumeManager{
		ObjectMeta: metav1.ObjectMeta{
			Name: "volume1",
		},
	})
	require.NotNil(t, volume)
	require.Nil(t, err)

	<-counter

	volumeList, err := volumeManagerClient.List(metav1.ListOptions{})
	require.NotNil(t, volumeList)
	require.Nil(t, err)

	// Verify there's just 1 object.
	require.Equal(t, 1, len(volumeList.Items))

	// Assert all of them were called.
	require.True(t, hook.addCalled)
}
