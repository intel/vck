package fake

import (
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"

	"github.com/NervanaSystems/kube-controllers-go/pkg/crd"
)

// ClientImpl is a fake implementation of crd.Client
type ClientImpl struct {
	CustomResourceImpl     *CustomResourceImpl
	CustomResourceListImpl *CustomResourceListImpl
	Error                  string
}

// returns a fake RESTClient.
// TODO Not used in unit tests, returns nil
func (c *ClientImpl) RESTClient() rest.Interface {
	return nil
}

// Create creates the supplied CRD.
func (c *ClientImpl) Create(cr crd.CustomResource) (e error) {
	if c.Error != "" {
		e = fmt.Errorf(c.Error)
	}
	return
}

// Get retrieves the CRD from the Kubernetes API server.
func (c *ClientImpl) Get(namespace string, name string) (result runtime.Object, e error) {
	if c.Error != "" {
		e = fmt.Errorf(c.Error)
	}
	result = c.CustomResourceImpl
	return
}

// Update updates the CRD on the Kubernetes API server.
func (c *ClientImpl) Update(cr crd.CustomResource) (result runtime.Object, e error) {
	if c.Error != "" {
		e = fmt.Errorf(c.Error)
	}
	result = c.CustomResourceImpl
	return
}

// Delete deletes the CRD from the Kubernetes API server.
func (c *ClientImpl) Delete(namespace string, name string) (e error) {
	if c.Error != "" {
		e = fmt.Errorf(c.Error)
	}
	return
}

// Validate validates a custom resource against a json schema.
// Returns nil if object adheres to the schema.
func (c *ClientImpl) Validate(cr crd.CustomResource) (e error) {
	if c.Error != "" {
		e = fmt.Errorf(c.Error)
	}
	return
}

func (c *ClientImpl) List(namespace string, labels map[string]string) (result runtime.Object, e error) {
	if c.Error != "" {
		e = fmt.Errorf(c.Error)
	}
	result = c.CustomResourceListImpl
	return
}
