package resource

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"encoding/json"
	"github.com/IntelAI/vck/pkg/resource/reify"
	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
)

type genericClient struct {
	resource           dynamic.ResourceInterface
	resourcePluralForm string
	namespace          string
	templateFileName   string
	scheme             *runtime.Scheme
	groupversion       runtime.GroupVersioner
	reify              reify.ReifyInterface
}

// NewGenericClient returns a new horizontal pod autoscaler client.
func NewGenericClient(resource dynamic.ResourceInterface, templateFileName string, resourcePluralForm string, scheme *runtime.Scheme, groupversion runtime.GroupVersioner, reify reify.ReifyInterface) Client {
	return &genericClient{
		resource:           resource,
		resourcePluralForm: resourcePluralForm,
		templateFileName:   templateFileName,
		scheme:             scheme,
		groupversion:       groupversion,
		reify:              reify,
	}
}

func (c *genericClient) Reify(templateValues interface{}) ([]byte, error) {
	result, err := c.reify.Reify(c.templateFileName, templateValues)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (c *genericClient) Create(namespace string, templateValues interface{}) error {
	resourceBody, err := c.Reify(templateValues)
	if err != nil {
		return err
	}

	// Create an unstructured object from the reified template so that it can be passed to the client.
	object := &unstructured.Unstructured{}
	err = json.Unmarshal(resourceBody, object)
	if err != nil {
		return err
	}

	_, err = c.resource.Create(object)
	if err != nil {
		return err
	}

	return nil
}

func (c *genericClient) Delete(namespace, name string) error {
	return c.resource.Delete(name, &metav1.DeleteOptions{})
}

func (c *genericClient) Get(namespace, name string) (result runtime.Object, err error) {
	res, err := c.resource.Get(name, metav1.GetOptions{})
	result, err = c.scheme.ConvertToVersion(res, c.groupversion)
	if err != nil {
		return nil, err
	}
	return
}

func (c *genericClient) List(namespace string, labels map[string]string) (result []metav1.Object, err error) {
	opts := metav1.ListOptions{}

	list, err := c.resource.List(opts)
	if err != nil {
		glog.Infof("[generic_client] Got err while listing: %v", err)
		return []metav1.Object{}, err
	}

	object := list.(*unstructured.UnstructuredList)

	for _, item := range object.Items {
		// We need a copy of the item here because item has function scope whereas the copy below has a local scope.
		// Ex: When we iterate through items, the result list will only contain multiple copies of the last item in the list.
		itemCopy := item
		result = append(result, &itemCopy)
	}

	return
}

// Plural returns the plural form of the resource.
func (c *genericClient) Plural() string {
	return c.resourcePluralForm
}

func (c *genericClient) Update(object runtime.Object) (result runtime.Object, err error) {

	convertedObject := &unstructured.Unstructured{}
	err = c.scheme.Convert(object, convertedObject, c.resource)
	if err != nil {
		return
	}
	result, err = c.resource.Update(convertedObject)
	return
}
