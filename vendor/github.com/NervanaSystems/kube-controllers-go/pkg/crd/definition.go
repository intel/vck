package crd

import (
	"fmt"
	"time"

	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	extv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8serrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/wait"
)

// Handle aggregates a CRD definition with additional data for
// client side (de)serialization.
type Handle struct {
	SchemaGroupVersion schema.GroupVersion
	Definition         *extv1beta1.CustomResourceDefinition
	ResourceType       runtime.Object
	ResourceListType   runtime.Object
	Plural             string
	SchemaURL          string
}

// New returns a new CRD Handle.
func New(
	resourceType runtime.Object,
	resourceListType runtime.Object,
	group string,
	version string,
	kind string,
	singular string,
	plural string,
	scope extv1beta1.ResourceScope,
	schemaURL string,
) *Handle {
	definition := &extv1beta1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s.%s", plural, group),
		},
		Spec: extv1beta1.CustomResourceDefinitionSpec{
			Group:   group,
			Version: version,
			Scope:   scope,
			Names: extv1beta1.CustomResourceDefinitionNames{
				Kind:     kind,
				Singular: singular,
				Plural:   plural,
			},
		},
	}

	return &Handle{
		SchemaGroupVersion: schema.GroupVersion{Group: group, Version: version},
		Definition:         definition,
		ResourceType:       resourceType,
		ResourceListType:   resourceListType,
		Plural:             plural,
		SchemaURL:          schemaURL,
	}
}

func (h *Handle) resourceName() string {
	return fmt.Sprintf("%s.%s", h.Definition.Spec.Names.Plural, h.Definition.Spec.Group)
}

// WriteDefinition writes the supplied CRD to the Kubernetes API server
// using the supplied client set.
func WriteDefinition(clientset apiextensionsclient.Interface, h *Handle) error {
	_, err := clientset.ApiextensionsV1beta1().CustomResourceDefinitions().Create(h.Definition)
	if err != nil {
		return err
	}

	var crd *apiextensionsv1beta1.CustomResourceDefinition
	// Wait for CRD to be established.
	err = wait.Poll(500*time.Millisecond, 60*time.Second, func() (bool, error) {
		crd, err = clientset.ApiextensionsV1beta1().CustomResourceDefinitions().Get(h.resourceName(), metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		for _, cond := range crd.Status.Conditions {
			switch cond.Type {
			case apiextensionsv1beta1.Established:
				if cond.Status == apiextensionsv1beta1.ConditionTrue {
					return true, err
				}
			case apiextensionsv1beta1.NamesAccepted:
				if cond.Status == apiextensionsv1beta1.ConditionFalse {
					fmt.Printf("Name conflict: %v\n", cond.Reason)
				}
			}
		}
		return false, err
	})
	if err != nil {
		deleteErr := DeleteDefinition(clientset, h)
		if deleteErr != nil {
			return k8serrors.NewAggregate([]error{err, deleteErr})
		}
		return err
	}

	// Update the definition in the supplied handle.
	h.Definition = crd

	return nil
}

// DeleteDefinition removes the supplied CRD to the Kubernetes API server
// using the supplied client set.
func DeleteDefinition(clientset apiextensionsclient.Interface, h *Handle) error {
	return clientset.ApiextensionsV1beta1().CustomResourceDefinitions().Delete(h.Definition.Name, nil)
}
