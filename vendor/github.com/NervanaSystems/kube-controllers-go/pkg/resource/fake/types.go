package fake

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/NervanaSystems/kube-controllers-go/pkg/states"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// Subresource implements a fake subresource
type Subresource struct {
	metav1.TypeMeta
	metav1.ObjectMeta

	SpecState   states.State
	StatusState states.State
	Ephemeral   bool
	Plural      string
	Lifecycle   string
}

// +k8s:deepcopy-gen=false
// SubresourceClient implements resource.Client
type SubresourceClient struct {
	// Use an interface so we can trigger a runtime.Object error
	// using a metav1.ObjectMeta that doesn't have a deepcopy_generated
	Subresource metav1.Object
	Error       string
	PluralValue string
}

// Reify returns an error - not implemented. Currently not used in unit tests
// TODO should return a valid bytpe array for unit tests
func (c *SubresourceClient) Reify(templateValues interface{}) ([]byte, error) {
	return nil, fmt.Errorf("not implemented")
}

// Create defines a fake resource.Client
func (c *SubresourceClient) Create(namespace string, templateValues interface{}) (e error) {
	if c.Error != "" {
		e = fmt.Errorf(c.Error)
	}
	return
}

// Delete deletes a fake resource.Client
func (c *SubresourceClient) Delete(namespace, name string) (e error) {
	if c.Error != "" {
		e = fmt.Errorf(c.Error)
	}
	return
}

// Get returns a fake runtime.Object
func (c *SubresourceClient) Get(namespace, name string) (result runtime.Object, e error) {
	if c.Error != "" {
		e = fmt.Errorf(c.Error)
	} else {
		sub := c.Subresource
		result = sub.(runtime.Object)
	}
	return
}

// List returns an array of fake metav1.Objects or error
func (c *SubresourceClient) List(namespace string, labels map[string]string) (result []metav1.Object, e error) {
	if c.Error != "" {
		e = fmt.Errorf(c.Error)
	} else {
		result = []metav1.Object{c.Subresource}
	}
	return
}

// IsFailed returns true if the resource is in a Failed state
func (c *SubresourceClient) IsFailed(namespace string, name string) bool {
	if c.Subresource.(*Subresource).StatusState == states.Failed {
		return true
	}
	return false
}

// IsEphemeral returns whether this Resource can be safely deleted and recreated
func (c *SubresourceClient) IsEphemeral() bool {
	return c.Subresource.(*Subresource).Ephemeral
}

// Plural returns the plural name of the fake resource.Client
func (c *SubresourceClient) Plural() (plural string) {
	return c.PluralValue
}

// GetStatusState returns the current status of the subresource.
func (c *SubresourceClient) GetStatusState(runtime.Object) states.State {
	return c.Subresource.(*Subresource).StatusState
}
