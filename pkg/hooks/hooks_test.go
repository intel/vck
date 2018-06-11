package hooks

import (
	vckv1 "github.com/IntelAI/vck/pkg/apis/vck/v1"
	vckv1_fake "github.com/IntelAI/vck/pkg/client/clientset/versioned/fake"
	"github.com/IntelAI/vck/pkg/handlers"
	"github.com/IntelAI/vck/pkg/states"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

type testDataHandler struct {
	addCalled    bool
	deleteCalled bool
	sourceType   vckv1.DataSourceType
}

func (tdh *testDataHandler) OnAdd(namespace string, vc vckv1.VolumeConfig, controllerRef metav1.OwnerReference) vckv1.Volume {
	tdh.addCalled = true
	return vckv1.Volume{}
}

func (tdh *testDataHandler) OnDelete(namespace string, vc vckv1.VolumeConfig, vStatus vckv1.Volume, controllerRef metav1.OwnerReference) {
	tdh.deleteCalled = true
}

func (tdh *testDataHandler) GetSourceType() vckv1.DataSourceType {
	return tdh.sourceType
}

func TestHook(t *testing.T) {

	// Create a fake CR client
	fakeClient := vckv1_fake.NewSimpleClientset()
	namespace := "test"

	// Test case 1, make sure it passes with one volume config.
	// Data handler's add should be called

	var s3SourceType vckv1.DataSourceType = "S3"
	fakeDataHandler := &testDataHandler{sourceType: s3SourceType}

	hook := NewVolumeManagerHooks(fakeClient.VckV1().VolumeManagers(namespace), []handlers.DataHandler{fakeDataHandler})

	// Create a fake vck CR
	volumeManager := &vckv1.VolumeManager{
		ObjectMeta: metav1.ObjectMeta{
			Name: "volumeManager",
		},
		Status: vckv1.VolumeManagerStatus{
			State:   states.Pending,
			Message: "Beginning sub-resource deployment",
		},
		Spec: vckv1.VolumeManagerSpec{
			VolumeConfigs: []vckv1.VolumeConfig{
				{
					SourceType: s3SourceType,
				},
			},
			State: states.Pending,
		},
	}

	// Create the CR using the fake client
	volumeManager, err := fakeClient.VckV1().VolumeManagers(namespace).Create(volumeManager)
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

	hook = NewVolumeManagerHooks(fakeClient.VckV1().VolumeManagers(namespace), []handlers.DataHandler{fakeDataHandler})

	// Add it
	hook.add(volumeManager)

	// Assert things.
	require.False(t, fakeDataHandler.addCalled)
	require.False(t, fakeDataHandler.deleteCalled)

	// Test case 3: Create a CR with an invalid spec state
	// It should not call any method in the handler
	// The status of the CR should be set to Failed too
	fakeClient = vckv1_fake.NewSimpleClientset()
	s3SourceType = "s3"
	fakeDataHandler = &testDataHandler{sourceType: s3SourceType}
	hook = NewVolumeManagerHooks(fakeClient.VckV1().VolumeManagers(namespace), []handlers.DataHandler{fakeDataHandler})

	volumeManager.Spec.State = states.Failed

	// Create the CR using the fake client
	volumeManager, err = fakeClient.VckV1().VolumeManagers(namespace).Create(volumeManager)
	require.NotNil(t, volumeManager)
	require.Nil(t, err)

	hook.add(volumeManager)

	// Assert things.
	require.False(t, fakeDataHandler.addCalled)
	require.False(t, fakeDataHandler.deleteCalled)

	// Get the CR
	volumeManager, err = fakeClient.VckV1().VolumeManagers(namespace).Get(volumeManager.Name, metav1.GetOptions{})
	require.NotNil(t, volumeManager)
	require.Nil(t, err)

	require.Equal(t, states.Failed, volumeManager.Status.State)
}
