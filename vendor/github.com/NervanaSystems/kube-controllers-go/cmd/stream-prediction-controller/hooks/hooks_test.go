package hooks

import (
	"fmt"
	"testing"

	extv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	crv1 "github.com/NervanaSystems/kube-controllers-go/cmd/stream-prediction-controller/apis/cr/v1"
	"github.com/NervanaSystems/kube-controllers-go/pkg/crd"
	"github.com/NervanaSystems/kube-controllers-go/pkg/resource"
	"github.com/NervanaSystems/kube-controllers-go/pkg/states"
	"github.com/NervanaSystems/kube-controllers-go/pkg/util"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/rest"
)

type testResourceClient struct {
	createCalled   bool
	createWillFail bool
	deleteCalled   bool
}

func (trc *testResourceClient) Reify(templateData interface{}) ([]byte, error) {
	return nil, nil
}

func (trc *testResourceClient) Create(namespace string, templateData interface{}) error {
	trc.createCalled = true

	if trc.createWillFail {
		return fmt.Errorf("Resource client creation failed on purpose")
	}

	return nil
}

func (trc *testResourceClient) Delete(namespace string, name string) error {
	trc.deleteCalled = true
	return nil
}

func (trc *testResourceClient) Get(namespace, name string) (result runtime.Object, err error) {
	return result, err
}

func (trc *testResourceClient) List(namespace string, labels map[string]string) (result []metav1.Object, err error) {
	return
}

func (trc *testResourceClient) Plural() string {
	return "fakePlural"
}

func (trc *testResourceClient) IsFailed(namespace string, name string) bool {
	return false
}

func (trc *testResourceClient) IsEphemeral() bool {
	return true
}

func (trc *testResourceClient) GetStatusState(obj runtime.Object) states.State {
	return states.Running
}

type testCRDClient struct{}

func (t *testCRDClient) Create(crd crd.CustomResource) error { return nil }

func (t *testCRDClient) Get(namespace string, name string) (runtime.Object, error) {
	return &crv1.StreamPrediction{}, nil
}

func (t *testCRDClient) Update(crd crd.CustomResource) (runtime.Object, error) {
	return &crv1.StreamPrediction{}, nil
}

func (t *testCRDClient) Delete(namespace string, name string) error { return nil }
func (t *testCRDClient) Validate(crd crd.CustomResource) error      { return nil }
func (t *testCRDClient) RESTClient() rest.Interface                 { return nil }
func (t *testCRDClient) List(namespace string, labels map[string]string) (runtime.Object, error) {
	return &crv1.StreamPredictionList{Items: []crv1.StreamPrediction{}}, nil
}

func TestStreampredictionHooks(t *testing.T) {
	sp := &crv1.StreamPrediction{
		ObjectMeta: metav1.ObjectMeta{
			Name: "stream-foobar",
		},
		Status: crv1.StreamPredictionStatus{
			State:   states.Pending,
			Message: "Created, not processed",
		},
		Spec: crv1.StreamPredictionSpec{},
	}
	//
	// First test, make sure the success case pass.
	// Both resources should be created and delete should not be called.
	//
	foo := &testResourceClient{createWillFail: false}
	bar := &testResourceClient{createWillFail: false}

	hooks := NewStreamPredictionHooks(&testCRDClient{}, []resource.Client{foo, bar})

	hooks.Add(sp)

	assert.True(t, foo.createCalled)
	assert.True(t, bar.createCalled)
	assert.False(t, foo.deleteCalled)
	assert.False(t, bar.deleteCalled)

	//
	// Second test, make sure that if the first resource fails, no other resources
	// wil be created. Both resources should be attempted to be deleted.
	//
	foo = &testResourceClient{createWillFail: true}
	bar = &testResourceClient{createWillFail: false}

	hooks = NewStreamPredictionHooks(&testCRDClient{}, []resource.Client{foo, bar})

	hooks.Add(sp)

	assert.True(t, foo.createCalled)
	assert.False(t, bar.createCalled)

	//
	// Third test, if the second resource fails. The first one should have been
	// attempted to be created. Both resources should be deleted.
	//
	foo = &testResourceClient{createWillFail: false}
	bar = &testResourceClient{createWillFail: true}

	hooks = NewStreamPredictionHooks(&testCRDClient{}, []resource.Client{foo, bar})

	hooks.Add(sp)

	assert.True(t, foo.createCalled)
	assert.True(t, bar.createCalled)

	// Update Tests
	// Check valid transitions
	// Transition from:
	// 1. Running -> Completed
	// In this case, all the resources should get undeployed.
	oldCRD := &crv1.StreamPrediction{
		Status: crv1.StreamPredictionStatus{
			State:   states.Running,
			Message: "Running, all resources are up",
		},
	}
	newCRD := &crv1.StreamPrediction{
		Spec: crv1.StreamPredictionSpec{
			State: states.Completed,
		},
		Status: crv1.StreamPredictionStatus{
			State:   states.Running,
			Message: "Running, all resources are up",
		},
	}

	foo = &testResourceClient{createWillFail: false}
	bar = &testResourceClient{createWillFail: false}

	hooks = NewStreamPredictionHooks(&testCRDClient{}, []resource.Client{foo, bar})

	hooks.Update(oldCRD, newCRD)

	assert.False(t, foo.createCalled)
	assert.False(t, bar.createCalled)
	assert.True(t, foo.deleteCalled)
	assert.True(t, bar.deleteCalled)

	// Invalid state change check
	// 2. Completed -> Failed
	// In this case, nothing should get called
	oldCRD = &crv1.StreamPrediction{
		Status: crv1.StreamPredictionStatus{
			State:   states.Completed,
			Message: "Completed the stream predict",
		},
	}
	newCRD = &crv1.StreamPrediction{
		Spec: crv1.StreamPredictionSpec{
			State: states.Failed,
		},
		Status: crv1.StreamPredictionStatus{
			State:   states.Completed,
			Message: "Completed the stream predict",
		},
	}

	foo = &testResourceClient{createWillFail: false}
	bar = &testResourceClient{createWillFail: false}

	hooks = NewStreamPredictionHooks(&testCRDClient{}, []resource.Client{foo, bar})

	hooks.Update(oldCRD, newCRD)

	assert.False(t, foo.createCalled)
	assert.False(t, bar.createCalled)
	assert.False(t, foo.deleteCalled)
	assert.False(t, bar.deleteCalled)
}

func TestSchemaValidation(t *testing.T) {
	config, err := util.BuildConfig("/go/src/github.com/NervanaSystems/kube-controllers-go/resources/config")
	assert.Nil(t, err)

	crdHandle := crd.New(
		&crv1.StreamPrediction{},
		&crv1.StreamPredictionList{},
		crv1.GroupName,
		crv1.Version,
		crv1.StreamPredictionResourceKind,
		crv1.StreamPredictionResourceSingular,
		crv1.StreamPredictionResourcePlural,
		extv1beta1.NamespaceScoped,
		"file:///go/src/github.com/NervanaSystems/kube-controllers-go/api/crd/stream-prediction-job-spec.json",
	)

	crdClient, err := crd.NewClient(*config, crdHandle)
	if err != nil {
		panic(err)
	}

	sp := &crv1.StreamPrediction{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "aipg.intel.com/v1",
			Kind:       "StreamPrediction",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "stream-20",
		},
		Spec: crv1.StreamPredictionSpec{
			NeonRepoSpec: crv1.NeonRepoSpec{
				RepoURL: "git@github.com:NervanaSystems/private-neon.git",
				Commit:  "v1.8.2",
			},
			SecuritySpec: crv1.SecuritySpec{
				PresignedToken: "95fcbe0cfe747b867655a243cee330",
				JWTToken:       "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdHJlYW1faWQiOjEwfQ.JxxqL8-6OV4xfQmy4dGRis3QSRuTJH2kattCfLHGKwA",
			},
			StreamDataSpec: crv1.StreamDataSpec{
				ModelPRM:         "/code/model.prm",
				ModelPath:        "s3://helium-joboutput-dev/integration/20dec8c3e38e2804888f252ef281121b/51/model.prm",
				DatasetPath:      "None",
				ExtraFilename:    "None",
				CustomCodeURL:    "None",
				CustomCommit:     "None",
				AWSPath:          "krypton-logs-dev/integration",
				AWSDefaultRegion: "us-west-1",
				StreamID:         20,
				StreamName:       "stream-20",
			},
			KryptonRepoSpec: crv1.KryptonRepoSpec{
				RepoURL:      "git@github.com:NervanaSystems/krypton.git",
				Commit:       "master",
				Image:        "nervana/krypton:master",
				SidecarImage: "nervana/krypton-sidecar:master",
			},
			State: "Pending",
		},
		Status: crv1.StreamPredictionStatus{
			State:   "Pending",
			Message: "Created, not processed",
		},
	}

	assert.Nil(t, crdClient.Validate(sp))
}
