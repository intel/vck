package resource

import (
	"fmt"
	"net/http"

	"github.com/golang/glog"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"

	"github.com/NervanaSystems/kube-controllers-go/pkg/resource/reify"
	"github.com/NervanaSystems/kube-controllers-go/pkg/states"
)

type hpaClient struct {
	globalTemplateValues GlobalTemplateValues
	restClient           rest.Interface
	resourcePluralForm   string
	templateFileName     string
}

// NewHPAClient returns a new horizontal pod autoscaler client.
func NewHPAClient(globalTemplateValues GlobalTemplateValues, clientSet *kubernetes.Clientset, templateFileName string) Client {
	return &hpaClient{
		globalTemplateValues: globalTemplateValues,
		restClient:           clientSet.AutoscalingV1().RESTClient(),
		resourcePluralForm:   "horizontalpodautoscalers",
		templateFileName:     templateFileName,
	}
}

func (c *hpaClient) Reify(templateValues interface{}) ([]byte, error) {
	result, err := reify.Reify(c.templateFileName, templateValues, c.globalTemplateValues)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (c *hpaClient) Create(namespace string, templateValues interface{}) error {
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

func (c *hpaClient) Delete(namespace, name string) error {
	request := c.restClient.Delete().
		Namespace(namespace).
		Resource(c.resourcePluralForm).
		Name(name)

	glog.Infof("[DEBUG] delete resource URL: %s", request.URL())

	return request.Do().Error()
}

func (c *hpaClient) Get(namespace, name string) (result runtime.Object, err error) {
	result = &autoscalingv1.HorizontalPodAutoscaler{}
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

func (c *hpaClient) List(namespace string, labels map[string]string) (result []metav1.Object, err error) {
	list := &autoscalingv1.HorizontalPodAutoscalerList{}
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
		hpaCopy := item
		result = append(result, &hpaCopy)
	}

	return
}

func (c *hpaClient) IsEphemeral() bool {
	return true
}

func (c *hpaClient) Plural() string {
	return c.resourcePluralForm
}

func (c *hpaClient) IsFailed(namespace string, name string) bool {
	return false
}

func (c *hpaClient) GetStatusState(obj runtime.Object) states.State {
	// TODO(CD): Detect Pending and Failed states. Completed doesn't make sense for this type.
	return states.Running
}
