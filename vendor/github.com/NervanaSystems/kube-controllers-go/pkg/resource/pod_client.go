package resource

import (
	"fmt"
	"net/http"

	"github.com/golang/glog"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apilabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"

	"github.com/NervanaSystems/kube-controllers-go/pkg/resource/reify"
	"github.com/NervanaSystems/kube-controllers-go/pkg/states"
)

type podClient struct {
	globalTemplateValues GlobalTemplateValues
	restClient           rest.Interface
	resourcePluralForm   string
	templateFileName     string
}

// NewPodClient returns a new pod client.
func NewPodClient(globalTemplateValues GlobalTemplateValues, clientSet *kubernetes.Clientset, templateFileName string) Client {
	return &podClient{
		globalTemplateValues: globalTemplateValues,
		restClient:           clientSet.CoreV1().RESTClient(),
		resourcePluralForm:   "pods",
		templateFileName:     templateFileName,
	}
}

func (c *podClient) Reify(templateValues interface{}) ([]byte, error) {
	result, err := reify.Reify(c.templateFileName, templateValues, c.globalTemplateValues)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (c *podClient) Create(namespace string, templateValues interface{}) error {
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

func (c *podClient) Delete(namespace, name string) error {
	request := c.restClient.Delete().
		Namespace(namespace).
		Resource(c.resourcePluralForm).
		Name(name)

	glog.Infof("[DEBUG] delete resource URL: %s", request.URL())

	return request.Do().Error()
}

func (c *podClient) Get(namespace, name string) (result runtime.Object, err error) {
	result = &corev1.Pod{}
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

func (c *podClient) List(namespace string, labels map[string]string) (result []metav1.Object, err error) {
	list := &corev1.PodList{}

	opts := metav1.ListOptions{}
	if len(labels) > 0 {
		selector := apilabels.SelectorFromSet(apilabels.Set(labels))
		opts = metav1.ListOptions{LabelSelector: selector.String()}
	}

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
		podCopy := item
		result = append(result, &podCopy)
	}

	return
}

func (c *podClient) IsEphemeral() bool {
	return true
}

func (c *podClient) Plural() string {
	return c.resourcePluralForm
}

func (c *podClient) IsFailed(namespace string, name string) bool {
	p, err := c.Get(namespace, name)
	if err != nil {
		return false
	}
	return c.isFailed(p)
}

func (c *podClient) isFailed(obj runtime.Object) bool {
	pod, ok := obj.(*corev1.Pod)
	if !ok {
		panic("object was not a *corev1.Pod")
	}
	if pod.Status.Phase == corev1.PodFailed {
		return true
	}
	for _, status := range pod.Status.ContainerStatuses {
		if !status.Ready && status.RestartCount > 0 {
			return true
		}
		// Indicates that a container in the pod was terminated with a non-zero exit code
		if !status.Ready && status.State.Terminated != nil && status.State.Terminated.ExitCode > 0 {
			return true
		}
	}
	return false
}

func (c *podClient) GetStatusState(obj runtime.Object) states.State {
	if c.isFailed(obj) {
		return states.Failed
	}

	// TODO(CD): Detect Pending, Completed and Failed states.
	return states.Running
}
