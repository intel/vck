package fake

import (
	"encoding/json"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/NervanaSystems/kube-controllers-go/pkg/states"
	"k8s.io/apimachinery/pkg/runtime"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// CustomResourceImpl implements crd.CustomResource
type CustomResourceImpl struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	SpecState         states.State `json:"spec"`
	StatusState       states.State `json:"status,omitempty"`
}

// Name returns objectMeta.Name
func (c *CustomResourceImpl) Name() string {
	return c.ObjectMeta.Name
}

// Namespace returns objectMeta.Namespace
func (c *CustomResourceImpl) Namespace() string {
	return c.ObjectMeta.Namespace
}

// JSON returns json representation
func (c *CustomResourceImpl) JSON() (string, error) {
	data, err := json.Marshal(c)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// GetSpecState returns spec.state
func (c *CustomResourceImpl) GetSpecState() states.State {
	return c.SpecState
}

// GetStatusState returns spec.status
func (c *CustomResourceImpl) GetStatusState() states.State {
	return c.StatusState
}

// SetStatusStateWithMessage TBD store message
func (c *CustomResourceImpl) SetStatusStateWithMessage(state states.State, message string) {
	c.StatusState = state
}

// CustomResourceListImpl implements crd.CustomResource for the List method
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type CustomResourceListImpl struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:",inline"`
	Items           []CustomResourceImpl `json:"items"`
}

func (c *CustomResourceListImpl) GetItems() []runtime.Object {
	var result []runtime.Object
	for _, item := range c.Items {
		itemCopy := item
		result = append(result, &itemCopy)
	}
	return result
}
