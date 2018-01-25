package resource

import (
	"fmt"
	"net/http"

	"github.com/golang/glog"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"

	"github.com/NervanaSystems/kube-controllers-go/pkg/resource/reify"
	"github.com/NervanaSystems/kube-controllers-go/pkg/states"
)

type jobClient struct {
	globalTemplateValues GlobalTemplateValues
	restClient           rest.Interface
	k8sClientset         *kubernetes.Clientset
	resourcePluralForm   string
	templateFileName     string
}

// NewJobClient returns a new generic resource client.
func NewJobClient(globalTemplateValues GlobalTemplateValues, clientSet *kubernetes.Clientset, templateFileName string) Client {
	return &jobClient{
		globalTemplateValues: globalTemplateValues,
		restClient:           clientSet.BatchV1().RESTClient(),
		k8sClientset:         clientSet,
		resourcePluralForm:   "jobs",
		templateFileName:     templateFileName,
	}
}

func (c *jobClient) Reify(templateValues interface{}) ([]byte, error) {
	result, err := reify.Reify(c.templateFileName, templateValues, c.globalTemplateValues)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (c *jobClient) Create(namespace string, templateValues interface{}) error {
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

func (c *jobClient) Delete(namespace, name string) error {
	deletePolicy := metav1.DeletePropagationForeground
	request := c.restClient.Delete().
		Namespace(namespace).
		Resource(c.resourcePluralForm).
		Name(name).
		Body(&metav1.DeleteOptions{
			PropagationPolicy: &deletePolicy,
		})

	glog.Infof("[DEBUG] delete resource URL: %s", request.URL())

	return request.Do().Error()
}

func (c *jobClient) Get(namespace, name string) (result runtime.Object, err error) {
	result = &batchv1.Job{}
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

func (c *jobClient) List(namespace string, labels map[string]string) (result []metav1.Object, err error) {
	list := &batchv1.JobList{}
	opts := metav1.ListOptions{}
	err = c.restClient.Get().
		Namespace(namespace).
		Resource(c.resourcePluralForm).
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(list)

	if err != nil {
		return result, err
	}

	for _, item := range list.Items {
		// We need a copy of the item here because item has function scope whereas the copy below has a local scope.
		// Ex: When we iterate through items, the result list will only contain multiple copies of the last item in the list.
		jobCopy := item
		result = append(result, &jobCopy)
	}

	return
}

func (c *jobClient) Plural() string {
	return c.resourcePluralForm
}

func (c *jobClient) IsFailed(namespace string, name string) bool {

	obj, err := c.Get(namespace, name)
	if err != nil {
		return false
	}

	return c.isFailed(obj)
}

func (c *jobClient) isFailed(obj runtime.Object) bool {
	job, ok := obj.(*batchv1.Job)
	if !ok {
		panic("Object was not a *batchv1.Job")
	}

	// We need to check the pod status before job status as the job status is not set if a container in the pod is in an error state.

	// If the job is not in a failed state we inspect whether the
	// containers controlled by the job are healthy.
	// This is required because the definition of pod failure in kubernetes is
	// strict. The pod is considered failed iff all containers in the pod have
	// terminated, and at least one container has terminated in a failure (exited
	// with a non-zero exit code or was stopped by the system).
	podClient := NewPodClient(GlobalTemplateValues{}, c.k8sClientset, "")

	// List all the pods with the same labels as the job and check if
	// they have failed.
	podList, err := podClient.List(job.ObjectMeta.Namespace, job.ObjectMeta.Labels)
	if err != nil {
		return false
	}

	for _, pod := range podList {
		if podClient.IsFailed(pod.GetNamespace(), pod.GetName()) {
			return true
		}
	}

	conditions := job.Status.Conditions
	if len(conditions) == 0 {
		return false
	}
	latestCondition := conditions[0]
	for i := range conditions {
		time1 := &latestCondition.LastTransitionTime
		time2 := &conditions[i].LastTransitionTime
		if time1.Before(time2) {
			latestCondition = conditions[i]
		}
	}

	if latestCondition.Type == batchv1.JobFailed {
		return true
	}

	return false
}

func (c *jobClient) IsEphemeral() bool {
	return false
}

func (c *jobClient) isCompleted(obj runtime.Object) bool {

	job, ok := obj.(*batchv1.Job)
	if !ok {
		panic("Object was not a *batchv1.Job")
	}

	conditions := job.Status.Conditions
	if len(conditions) == 0 {
		return false
	}
	latestCondition := conditions[0]
	for i := range conditions {
		time1 := &latestCondition.LastTransitionTime
		time2 := &conditions[i].LastTransitionTime
		if time1.Before(time2) {
			latestCondition = conditions[i]
		}
	}

	if latestCondition.Type == batchv1.JobComplete {
		return true
	}
	return false
}

func (c *jobClient) GetStatusState(obj runtime.Object) states.State {
	// TODO(CD): Detect Pending, Completed and Failed states.

	if c.isCompleted(obj) {
		return states.Completed
	}

	if c.isFailed(obj) {
		return states.Failed
	}

	return states.Running
}
