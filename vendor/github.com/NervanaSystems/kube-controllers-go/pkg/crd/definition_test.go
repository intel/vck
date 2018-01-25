package crd

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	extv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1beta1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	discovery "k8s.io/client-go/discovery"
	rest "k8s.io/client-go/rest"
)

func TestWriteDefinitionOK(t *testing.T) {
	clientSet := newFakeClientSet()

	var firstCRD *extv1beta1.CustomResourceDefinition

	clientSet.fakeApiextensions.crd.CreateHandler = func(crd *extv1beta1.CustomResourceDefinition) (*extv1beta1.CustomResourceDefinition, error) {
		firstCRD = crd
		return crd, nil
	}

	clientSet.fakeApiextensions.crd.GetHandler = func(name string, options v1.GetOptions) (*extv1beta1.CustomResourceDefinition, error) {
		firstCRD.Status = extv1beta1.CustomResourceDefinitionStatus{
			Conditions: []extv1beta1.CustomResourceDefinitionCondition{
				{
					Type:   extv1beta1.Established,
					Status: extv1beta1.ConditionTrue,
					Reason: "Was set artificially by Create()",
				},
			},
			AcceptedNames: extv1beta1.CustomResourceDefinitionNames{
				Plural:     "TestCRDs",
				Singular:   "TestCRD",
				ShortNames: []string{"tcrd"},
				Kind:       "TestCRD",
				ListKind:   "TestCRDList",
			},
		}

		return firstCRD, nil
	}

	err := WriteDefinition(clientSet, New(
		&TestCRD{},
		&TestCRDList{},
		"test.intel.com",
		"v1",
		"TestCRD",
		"testCRD",
		"testCRDs",
		extv1beta1.NamespaceScoped,
		""))
	require.Nil(t, err)
	require.Equal(t, clientSet.fakeApiextensions.crd.CreateCount, 1)
}

func TestWriteDefinitionCreateError(t *testing.T) {
	clientSet := newFakeClientSet()

	clientSet.fakeApiextensions.crd.CreateHandler = func(crd *extv1beta1.CustomResourceDefinition) (*extv1beta1.CustomResourceDefinition, error) {
		return nil, fmt.Errorf("Some failure during creation")
	}

	err := WriteDefinition(clientSet, New(
		&TestCRD{},
		&TestCRDList{},
		"test.intel.com",
		"v1",
		"TestCRD",
		"testCRD",
		"testCRDs",
		extv1beta1.NamespaceScoped,
		""))
	require.NotNil(t, err)
}

func TestWriteDefinitionGetError(t *testing.T) {
	clientSet := newFakeClientSet()

	clientSet.fakeApiextensions.crd.GetHandler = func(name string, options v1.GetOptions) (*extv1beta1.CustomResourceDefinition, error) {
		return nil, fmt.Errorf("Some failure during get")
	}

	err := WriteDefinition(clientSet, New(
		&TestCRD{},
		&TestCRDList{},
		"test.intel.com",
		"v1",
		"TestCRD",
		"testCRD",
		"testCRDs",
		extv1beta1.NamespaceScoped,
		""))
	require.NotNil(t, err)
	require.Equal(t, clientSet.fakeApiextensions.crd.CreateCount, 1)
	require.Equal(t, clientSet.fakeApiextensions.crd.DeleteCount, 1)
}

func TestDeleteDefinitionOK(t *testing.T) {
	clientSet := newFakeClientSet()

	err := DeleteDefinition(clientSet, New(
		&TestCRD{},
		&TestCRDList{},
		"test.intel.com",
		"v1",
		"TestCRD",
		"testCRD",
		"testCRDs",
		extv1beta1.NamespaceScoped,
		""))
	require.Nil(t, err)
	require.Equal(t, clientSet.fakeApiextensions.crd.DeleteCount, 1)
}

func TestWriteDefinitionPoling(t *testing.T) {
	clientSet := newFakeClientSet()

	var firstCRD *extv1beta1.CustomResourceDefinition

	clientSet.fakeApiextensions.crd.CreateHandler = func(crd *extv1beta1.CustomResourceDefinition) (*extv1beta1.CustomResourceDefinition, error) {
		firstCRD = crd
		return crd, nil
	}

	retries := 3

	clientSet.fakeApiextensions.crd.GetHandler = func(name string, options v1.GetOptions) (*extv1beta1.CustomResourceDefinition, error) {
		if retries > 0 {
			retries--
			return firstCRD, nil
		}

		firstCRD.Status = extv1beta1.CustomResourceDefinitionStatus{
			Conditions: []extv1beta1.CustomResourceDefinitionCondition{
				{
					Type:   extv1beta1.Established,
					Status: extv1beta1.ConditionTrue,
					Reason: "Was set artificially by Create()",
				},
			},
			AcceptedNames: extv1beta1.CustomResourceDefinitionNames{
				Plural:     "TestCRDs",
				Singular:   "TestCRD",
				ShortNames: []string{"tcrd"},
				Kind:       "TestCRD",
				ListKind:   "TestCRDList",
			},
		}

		return firstCRD, nil
	}

	err := WriteDefinition(clientSet, New(
		&TestCRD{},
		&TestCRDList{},
		"test.intel.com",
		"v1",
		"TestCRD",
		"testCRD",
		"testCRDs",
		extv1beta1.NamespaceScoped,
		""))
	require.Nil(t, err)
	require.Equal(t, clientSet.fakeApiextensions.crd.CreateCount, 1)
}

func newFakeClientSet() *fakeClientSet {
	clientSet := &fakeClientSet{}
	clientSet.ApiextensionsV1beta1().CustomResourceDefinitions()
	return clientSet
}

type fakeClientSet struct {
	fakeApiextensions *fakeApiextensionsV1beta1
}

func (clientSet *fakeClientSet) Discovery() discovery.DiscoveryInterface { return nil }

func (clientSet *fakeClientSet) ApiextensionsV1beta1() apiextensionsv1beta1.ApiextensionsV1beta1Interface {
	if clientSet.fakeApiextensions == nil {
		clientSet.fakeApiextensions = &fakeApiextensionsV1beta1{}
	}
	return clientSet.fakeApiextensions
}

func (clientSet *fakeClientSet) Apiextensions() apiextensionsv1beta1.ApiextensionsV1beta1Interface {
	return nil
}

type fakeApiextensionsV1beta1 struct {
	crd *fakeCustomResourceDefinition
}

func (fakeApiextensions *fakeApiextensionsV1beta1) RESTClient() rest.Interface {
	return nil
}

func (fakeApiextensions *fakeApiextensionsV1beta1) CustomResourceDefinitions() apiextensionsv1beta1.CustomResourceDefinitionInterface {
	if fakeApiextensions.crd == nil {
		fakeApiextensions.crd = &fakeCustomResourceDefinition{}
	}

	return fakeApiextensions.crd
}

type fakeCustomResourceDefinition struct {
	CreateHandler           func(crd *extv1beta1.CustomResourceDefinition) (*extv1beta1.CustomResourceDefinition, error)
	CreateCount             int
	UpdateHandler           func(*extv1beta1.CustomResourceDefinition) (*extv1beta1.CustomResourceDefinition, error)
	UpdateCount             int
	UpdateStatusHandler     func(*extv1beta1.CustomResourceDefinition) (*extv1beta1.CustomResourceDefinition, error)
	UpdateStatusCount       int
	DeleteHandler           func(name string, options *v1.DeleteOptions) error
	DeleteCount             int
	DeleteCollectionHandler func(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	DeleteCollectionCount   int
	GetHandler              func(name string, options v1.GetOptions) (*extv1beta1.CustomResourceDefinition, error)
	GetCount                int
	ListHandler             func(opts v1.ListOptions) (*extv1beta1.CustomResourceDefinitionList, error)
	ListCount               int
	WatchHandler            func(opts v1.ListOptions) (watch.Interface, error)
	WatchCount              int
	PatchHandler            func(name string, pt types.PatchType, data []byte, subresources ...string) (result *extv1beta1.CustomResourceDefinition, err error)
	PatchCount              int
}

func (fcrd *fakeCustomResourceDefinition) Create(crd *extv1beta1.CustomResourceDefinition) (*extv1beta1.CustomResourceDefinition, error) {
	fcrd.CreateCount++
	if fcrd.CreateHandler != nil {
		return fcrd.CreateHandler(crd)
	}
	return nil, nil
}

func (fcrd *fakeCustomResourceDefinition) Update(crd *extv1beta1.CustomResourceDefinition) (*extv1beta1.CustomResourceDefinition, error) {
	fcrd.UpdateCount++
	if fcrd.UpdateHandler != nil {
		return fcrd.UpdateHandler(crd)
	}
	return nil, nil
}

func (fcrd *fakeCustomResourceDefinition) UpdateStatus(crd *extv1beta1.CustomResourceDefinition) (*extv1beta1.CustomResourceDefinition, error) {
	fcrd.UpdateStatusCount++
	if fcrd.UpdateStatusHandler != nil {
		return fcrd.UpdateStatusHandler(crd)
	}
	return nil, nil
}

func (fcrd *fakeCustomResourceDefinition) Delete(name string, options *v1.DeleteOptions) error {
	fcrd.DeleteCount++
	if fcrd.DeleteHandler != nil {
		return fcrd.DeleteHandler(name, options)
	}
	return nil
}

func (fcrd *fakeCustomResourceDefinition) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	fcrd.DeleteCollectionCount++
	if fcrd.DeleteCollectionHandler != nil {
		return fcrd.DeleteCollectionHandler(options, listOptions)
	}
	return nil
}

func (fcrd *fakeCustomResourceDefinition) Get(name string, options v1.GetOptions) (*extv1beta1.CustomResourceDefinition, error) {
	fcrd.GetCount++
	if fcrd.GetHandler != nil {
		return fcrd.GetHandler(name, options)
	}
	return nil, nil
}

func (fcrd *fakeCustomResourceDefinition) List(opts v1.ListOptions) (*extv1beta1.CustomResourceDefinitionList, error) {
	fcrd.ListCount++
	if fcrd.ListHandler != nil {
		return fcrd.ListHandler(opts)
	}
	return nil, nil
}

func (fcrd *fakeCustomResourceDefinition) Watch(opts v1.ListOptions) (watch.Interface, error) {
	fcrd.WatchCount++
	if fcrd.WatchHandler != nil {
		return fcrd.WatchHandler(opts)
	}
	return nil, nil
}

func (fcrd *fakeCustomResourceDefinition) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *extv1beta1.CustomResourceDefinition, err error) {
	fcrd.PatchCount++
	if fcrd.PatchHandler != nil {
		return fcrd.Patch(name, pt, data, subresources...)
	}
	return nil, nil
}
