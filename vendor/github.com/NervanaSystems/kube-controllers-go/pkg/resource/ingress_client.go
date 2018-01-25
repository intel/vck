package resource

import (
	"fmt"
	"net/http"

	"github.com/golang/glog"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"

	"github.com/NervanaSystems/kube-controllers-go/pkg/resource/reify"
	"github.com/NervanaSystems/kube-controllers-go/pkg/states"
)

type ingressClient struct {
	globalTemplateValues GlobalTemplateValues
	restClient           rest.Interface
	resourcePluralForm   string
	templateFileName     string
}

// NewIngressClient returns a new ingress client.
func NewIngressClient(globalTemplateValues GlobalTemplateValues, clientSet *kubernetes.Clientset, templateFileName string) Client {
	return &ingressClient{
		globalTemplateValues: globalTemplateValues,
		restClient:           clientSet.ExtensionsV1beta1().RESTClient(),
		resourcePluralForm:   "ingresses",
		templateFileName:     templateFileName,
	}
}

func (c *ingressClient) Reify(templateValues interface{}) ([]byte, error) {
	result, err := reify.Reify(c.templateFileName, templateValues, c.globalTemplateValues)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (c *ingressClient) Create(namespace string, templateValues interface{}) error {
	resourceBody, err := c.Reify(templateValues)
	if err != nil {
		return err
	}

	request := c.restClient.Post().
		Namespace(namespace).
		Resource(c.resourcePluralForm).
		Body(resourceBody)

	glog.Infof("[DEBUG] create resource URL: %s", request.URL())

	var statusCode int
	err = request.Do().StatusCode(&statusCode).Error()

	if err != nil {
		return err
	}
	if statusCode != http.StatusOK && statusCode != http.StatusCreated {
		return fmt.Errorf("unexpected status code (%d)", statusCode)
	}
	return nil
}

func (c *ingressClient) Delete(namespace, name string) error {
	request := c.restClient.Delete().
		Namespace(namespace).
		Resource(c.resourcePluralForm).
		Name(name)

	glog.Infof("[DEBUG] delete resource URL: %s", request.URL())

	return request.Do().Error()
}

func (c *ingressClient) Get(namespace, name string) (result runtime.Object, err error) {
	result = &v1beta1.Ingress{}
	opts := metav1.GetOptions{}
	err = c.restClient.Get().
		Namespace(namespace).
		Resource(c.resourcePluralForm).
		Name(name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)

	return result, err
}

func (c *ingressClient) List(namespace string, labels map[string]string) (result []metav1.Object, err error) {
	list := &v1beta1.IngressList{}
	opts := metav1.ListOptions{}
	err = c.restClient.Get().
		Namespace(namespace).
		Resource(c.resourcePluralForm).
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(list)

	if err != nil {
		return []metav1.Object{}, err
	}

	for _, item := range list.Items {
		// We need a copy of the item here because item has function scope whereas the copy below has a local scope.
		// Ex: When we iterate through items, the result list will only contain multiple copies of the last item in the list.
		ingressCopy := item
		result = append(result, &ingressCopy)
	}

	return
}

func (c *ingressClient) IsEphemeral() bool {
	return true
}

func (c *ingressClient) Plural() string {
	return c.resourcePluralForm
}

func (c *ingressClient) IsFailed(namespace string, name string) bool {
	return false
}

func (c *ingressClient) GetStatusState(obj runtime.Object) states.State {
	// TODO(CD): Detect Pending and Failed states. Completed doesn't make sense for this type.
	return states.Running
}
