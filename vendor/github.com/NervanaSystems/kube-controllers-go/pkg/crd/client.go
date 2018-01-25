package crd

import (
	"errors"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/rest"
	// Uncomment the following line to load the gcp plugin (only required to authenticate against GKE clusters).
	// _ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	"github.com/xeipuuv/gojsonschema"
)

const apiRoot = "/apis"

// Client is used to handle CRD operations.
type Client interface {
	Create(crd CustomResource) error
	Get(namespace string, name string) (runtime.Object, error)
	Update(crd CustomResource) (runtime.Object, error)
	Delete(namespace string, name string) error
	Validate(crd CustomResource) error
	RESTClient() rest.Interface
	List(namespace string, labels map[string]string) (runtime.Object, error)
}

type client struct {
	restClient rest.Interface
	handle     *Handle
}

// NewClient returns a new REST client wrapper for the supplied CRD handle.
func NewClient(config rest.Config, h *Handle) (Client, error) {
	// TODO(balajismaninam): move scheme building to register.go in crv1.
	// We can enable metav1.GetOptions and metav1.ListOptions after that.
	scheme := runtime.NewScheme()

	scheme.AddKnownTypes(h.SchemaGroupVersion, h.ResourceType, h.ResourceListType)
	metav1.AddToGroupVersion(scheme, h.SchemaGroupVersion)

	config.GroupVersion = &h.SchemaGroupVersion
	config.APIPath = apiRoot
	config.ContentType = runtime.ContentTypeJSON
	config.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: serializer.NewCodecFactory(scheme)}

	restClient, err := rest.RESTClientFor(&config)
	if err != nil {
		return nil, err
	}

	return &client{restClient, h}, nil
}

func (c *client) RESTClient() rest.Interface {
	return c.restClient
}

// Create creates the supplied CRD.
func (c *client) Create(crd CustomResource) error {
	if c.handle.SchemaURL != "" {
		if err := c.Validate(crd); err != nil {
			return err
		}
	}

	return c.restClient.Post().
		Namespace(crd.Namespace()).
		Resource(c.handle.Plural).
		Name(crd.Name()).
		Body(crd).
		Do().
		Error()
}

// Get retrieves the CRD from the Kubernetes API server.
func (c *client) Get(namespace string, name string) (runtime.Object, error) {
	// TODO(balajismaniam): Move scheme building to register.go in crv1 and
	// enable the usage of metav1.GetOptions{}.
	result := c.handle.ResourceType.DeepCopyObject()
	err := c.restClient.Get().
		Namespace(namespace).
		Resource(c.handle.Plural).
		Name(name).
		Do().
		Into(result)

	return result, err
}

// List retrieves the list of CRs from the API server
func (c *client) List(namespace string, labels map[string]string) (runtime.Object, error) {
	result := c.handle.ResourceListType.DeepCopyObject()
	err := c.restClient.Get().
		Namespace(namespace).
		Resource(c.handle.Plural).
		Do().
		Into(result)

	return result, err
}

// Update updates the CRD on the Kubernetes API server.
func (c *client) Update(crd CustomResource) (runtime.Object, error) {
	if c.handle.SchemaURL != "" {
		if err := c.Validate(crd); err != nil {
			return nil, err
		}
	}

	resp := c.restClient.Put().
		Namespace(crd.Namespace()).
		Resource(c.handle.Plural).
		Name(crd.Name()).
		Body(crd).
		Do()

	obj, err := resp.Get()
	if err != nil {
		return nil, err
	}

	return obj, resp.Error()
}

// Delete deletes the CRD from the Kubernetes API server.
func (c *client) Delete(namespace string, name string) error {
	return c.restClient.Delete().
		Namespace(namespace).
		Resource(c.handle.Plural).
		Name(name).
		Do().
		Error()
}

// Validate validates a custom resource against a json schema.
// Returns nil if object adheres to the schema.
func (c *client) Validate(cr CustomResource) error {
	if c.handle.SchemaURL == "" {
		return fmt.Errorf("Validate called without schema URL set")
	}

	schemaLoader := gojsonschema.NewReferenceLoader(c.handle.SchemaURL)

	json, err := cr.JSON()
	if err != nil {
		return err
	}

	documentLoader := gojsonschema.NewStringLoader(json)
	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return err
	}

	if !result.Valid() {
		errorOutput := "Invalid JSON: '" + json + "': "

		for _, desc := range result.Errors() {
			errorOutput = errorOutput + " - " + desc.String() + "\n"
		}

		return errors.New(errorOutput)
	}

	return nil
}
