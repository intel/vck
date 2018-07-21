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

package hooks

import (
	vckv1alpha1 "github.com/ppkube/vck/pkg/apis/vck/v1alpha1"
	vckv1alpha1_fake "github.com/ppkube/vck/pkg/client/clientset/versioned/fake"
	"github.com/ppkube/vck/pkg/handlers"
	"github.com/ppkube/vck/pkg/states"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

type testDataHandler struct {
	addCalled    bool
	deleteCalled bool
	sourceType   vckv1alpha1.DataSourceType
}

func (tdh *testDataHandler) OnAdd(namespace string, vc vckv1alpha1.VolumeConfig, controllerRef metav1.OwnerReference) vckv1alpha1.Volume {
	tdh.addCalled = true
	return vckv1alpha1.Volume{}
}

func (tdh *testDataHandler) OnDelete(namespace string, vc vckv1alpha1.VolumeConfig, vStatus vckv1alpha1.Volume, controllerRef metav1.OwnerReference) {
	tdh.deleteCalled = true
}

func (tdh *testDataHandler) GetSourceType() vckv1alpha1.DataSourceType {
	return tdh.sourceType
}

func TestHook(t *testing.T) {

	// Create a fake CR client
	fakeClient := vckv1alpha1_fake.NewSimpleClientset()
	namespace := "test"

	// Test case 1, make sure it passes with one volume config.
	// Data handler's add should be called

	var s3SourceType vckv1alpha1.DataSourceType = "S3"
	fakeDataHandler := &testDataHandler{sourceType: s3SourceType}

	hook := NewVolumeManagerHooks(fakeClient.VckV1alpha1().VolumeManagers(namespace), []handlers.DataHandler{fakeDataHandler})

	// Create a fake vck CR
	volumeManager := &vckv1alpha1.VolumeManager{
		ObjectMeta: metav1.ObjectMeta{
			Name: "volumeManager",
		},
		Status: vckv1alpha1.VolumeManagerStatus{
			State:   states.Pending,
			Message: "Beginning sub-resource deployment",
		},
		Spec: vckv1alpha1.VolumeManagerSpec{
			VolumeConfigs: []vckv1alpha1.VolumeConfig{
				{
					SourceType: s3SourceType,
				},
			},
			State: states.Pending,
		},
	}

	// Create the CR using the fake client
	volumeManager, err := fakeClient.VckV1alpha1().VolumeManagers(namespace).Create(volumeManager)
	require.NotNil(t, volumeManager)
	require.Nil(t, err)

	// Add it
	hook.add(volumeManager)

	// Assert things.
	require.True(t, fakeDataHandler.addCalled)
	require.False(t, fakeDataHandler.deleteCalled)

	// Test case 2: If a CR is created without a valid source type,
	// the add/delete should not get called.
	// TODO(ajay): should we error the CR in that case?
	s3SourceType = "foo"
	fakeDataHandler = &testDataHandler{sourceType: s3SourceType}

	hook = NewVolumeManagerHooks(fakeClient.VckV1alpha1().VolumeManagers(namespace), []handlers.DataHandler{fakeDataHandler})

	// Add it
	hook.add(volumeManager)

	// Assert things.
	require.False(t, fakeDataHandler.addCalled)
	require.False(t, fakeDataHandler.deleteCalled)

	// Test case 3: Create a CR with an invalid spec state
	// It should not call any method in the handler
	// The status of the CR should be set to Failed too
	fakeClient = vckv1alpha1_fake.NewSimpleClientset()
	s3SourceType = "s3"
	fakeDataHandler = &testDataHandler{sourceType: s3SourceType}
	hook = NewVolumeManagerHooks(fakeClient.VckV1alpha1().VolumeManagers(namespace), []handlers.DataHandler{fakeDataHandler})

	volumeManager.Spec.State = states.Failed

	// Create the CR using the fake client
	volumeManager, err = fakeClient.VckV1alpha1().VolumeManagers(namespace).Create(volumeManager)
	require.NotNil(t, volumeManager)
	require.Nil(t, err)

	hook.add(volumeManager)

	// Assert things.
	require.False(t, fakeDataHandler.addCalled)
	require.False(t, fakeDataHandler.deleteCalled)

	// Get the CR
	volumeManager, err = fakeClient.VckV1alpha1().VolumeManagers(namespace).Get(volumeManager.Name, metav1.GetOptions{})
	require.NotNil(t, volumeManager)
	require.Nil(t, err)

	require.Equal(t, states.Failed, volumeManager.Status.State)
}
