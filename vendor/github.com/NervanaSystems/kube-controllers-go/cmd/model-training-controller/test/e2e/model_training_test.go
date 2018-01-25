package e2e

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	apiv1 "k8s.io/api/core/v1"
	extv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"

	"github.com/segmentio/ksuid"
	"github.com/stretchr/testify/require"

	crv1 "github.com/NervanaSystems/kube-controllers-go/cmd/model-training-controller/apis/cr/v1"
	"github.com/NervanaSystems/kube-controllers-go/pkg/crd"
	"github.com/NervanaSystems/kube-controllers-go/pkg/states"
	"github.com/NervanaSystems/kube-controllers-go/pkg/util"
	batchv1 "k8s.io/api/batch/v1"
)

const NAMESPACE = "e2e-test"

func makeClients(t *testing.T) (crd.Client, *kubernetes.Clientset) {
	config, err := util.BuildConfig("/go/src/github.com/NervanaSystems/kube-controllers-go/resources/config")
	require.Nil(t, err)

	crdHandle := crd.New(
		&crv1.ModelTraining{},
		&crv1.ModelTrainingList{},
		crv1.GroupName,
		crv1.Version,
		crv1.ModelTrainingResourceKind,
		crv1.ModelTrainingResourceSingular,
		crv1.ModelTrainingResourcePlural,
		extv1beta1.NamespaceScoped,
		"",
	)

	crdClient, err := crd.NewClient(*config, crdHandle)
	require.Nil(t, err)
	require.NotNil(t, crdClient)

	k8sClient, err := kubernetes.NewForConfig(config)
	require.Nil(t, err)
	require.NotNil(t, k8sClient)

	return crdClient, k8sClient
}

func makeModelTraining(modelName string) *crv1.ModelTraining {
	return &crv1.ModelTraining{
		ObjectMeta: metav1.ObjectMeta{
			Name: modelName,
		},
		Spec: crv1.ModelTrainingSpec{
			JobID:    "fakeModelID",
			TenantID: "fakeTenantID",
			ContainerSpec: crv1.ContainerSpec{
				Image:        "fakeImage",
				SidecarImage: "fakeSideCarImage",
				NFLImage:     "fakeNFLImage",
			},
			SandboxS3URLPath: "fakeSandboxS3URL",
			VolumeS3URLs:     []crv1.S3URL{"s3:fakeVolumesS3URL1", "s3:fakeVolumesS3URL2"},
			MetricsURL:       "fakeMetricsURL",
			Repositories: []crv1.Repository{
				{
					Name:   "NAME1",
					URL:    "fakeGHURL1",
					Commit: "fakeCommit1",
				},
				{
					Name:   "NAME2",
					URL:    "fakeGHURL2",
					Commit: "fakeCommit2",
				},
				{
					Name:   "NAME3",
					URL:    "fakeGHURL3",
					Commit: "fakeCommit3",
				},
			},
			ContinuationS3URL: "s3:continuationS3URL",
			State:             states.Completed,
			ResourceSpec: crv1.ResourceSpec{
				Requests: map[string]resource.Quantity{
					"cpu":    resource.MustParse("1"),
					"memory": resource.MustParse("512M"),
					"alpha.kubernetes.io/nvidia-gpu": resource.MustParse("2"),
				},
			},
		},
		Status: crv1.ModelTrainingStatus{
			State:   states.Pending,
			Message: "Created, not processed",
		},
	}
}

func makePodWithLabels(podName string, labels map[string]string) *apiv1.Pod {
	var containers []apiv1.Container
	ctn := apiv1.Container{
		Name:  "fakecontainername",
		Image: "busybox",
	}
	containers = append(containers, ctn)

	return &apiv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:   podName,
			Labels: labels,
		},
		Spec: apiv1.PodSpec{
			Containers: containers,
		},
	}
}

func refresh(t *testing.T, local *crv1.ModelTraining, crdClient crd.Client) {
	modelName := local.Name()
	namespace := local.Namespace()
	*local = crv1.ModelTraining{}
	err := crdClient.RESTClient().Get().
		Resource(crv1.ModelTrainingResourcePlural).
		Namespace(namespace).
		Name(modelName).
		Do().
		Into(local)
	require.Nil(t, err)
}

func TestModelTraining(t *testing.T) {
	crdClient, k8sClient := makeClients(t)

	modelName := fmt.Sprintf("model%s", strings.ToLower(ksuid.New().String()))
	original := makeModelTraining(modelName)

	copy := &crv1.ModelTraining{}
	err := crdClient.RESTClient().Post().
		Resource(crv1.ModelTrainingResourcePlural).
		Namespace(NAMESPACE).
		Body(original).
		Do().
		Into(copy)

	if err == nil {
		t.Logf("Created model training job: %#v\n", copy)
	} else if apierrors.IsAlreadyExists(err) {
		t.Errorf("model training job already exists: %#v\n", copy)
	} else {
		t.Fatal(err)
	}

	// Check whether the job was created successfully
	refresh(t, copy, crdClient)
	testSpec(t, copy, &(original.Spec))

	// Check whether the job was processed.
	// In the Running state, all subresources should exist.
	checkModelTrainingState(t, copy, crdClient, modelName, k8sClient, NAMESPACE, states.Running, true)

	refresh(t, copy, crdClient)
	testSpec(t, copy, &(original.Spec))

	// Get the job and set it's status to Completed.
	job := &batchv1.Job{}
	job, err = k8sClient.BatchV1().
		Jobs(NAMESPACE).Get(modelName, metav1.GetOptions{})
	require.Nil(t, err)
	require.NotNil(t, job)

	jobConditions := job.Status.Conditions
	jobConditions = append(jobConditions, batchv1.JobCondition{
		Type:               batchv1.JobComplete,
		Status:             apiv1.ConditionTrue,
		LastProbeTime:      metav1.Now(),
		LastTransitionTime: metav1.Now(),
	})

	job.Status.Conditions = jobConditions
	completedJob, err := k8sClient.BatchV1().
		Jobs(NAMESPACE).UpdateStatus(job)
	require.Nil(t, err)
	require.NotNil(t, completedJob)
	require.Equal(t, completedJob.Status.Conditions[0].Type, batchv1.JobComplete)

	refresh(t, copy, crdClient)
	checkModelTrainingState(t, copy, crdClient, modelName, k8sClient, NAMESPACE, states.Completed, false)

	err = crdClient.Delete(NAMESPACE, modelName)
	require.Nil(t, err)

	modelTrainingList := &crv1.ModelTrainingList{}
	require.Nil(t, crdClient.RESTClient().
		Get().
		Resource(crv1.ModelTrainingResourcePlural).
		Do().
		Into(modelTrainingList))
	require.Equal(t, len(modelTrainingList.Items), 0)
}

func TestModelTrainingPodFailed(t *testing.T) {
	crdClient, k8sClient := makeClients(t)

	modelName := fmt.Sprintf("model%s", strings.ToLower(ksuid.New().String()))
	original := makeModelTraining(modelName)

	copy := &crv1.ModelTraining{}
	err := crdClient.RESTClient().Post().
		Resource(crv1.ModelTrainingResourcePlural).
		Namespace(NAMESPACE).
		Body(original).
		Do().
		Into(copy)

	if err == nil {
		t.Logf("Created model training job: %#v\n", copy)
	} else if apierrors.IsAlreadyExists(err) {
		t.Errorf("model training job already exists: %#v\n", copy)
	} else {
		t.Fatal(err)
	}

	// Check whether the job was created successfully
	refresh(t, copy, crdClient)
	testSpec(t, copy, &(original.Spec))

	// Check whether the job was processed.
	// In the Running state, all subresources should exist.
	checkModelTrainingState(t, copy, crdClient, modelName, k8sClient, NAMESPACE, states.Running, true)

	refresh(t, copy, crdClient)
	testSpec(t, copy, &(original.Spec))

	// Update the job condition to ReplicaFailure.
	job := &batchv1.Job{}
	job, err = k8sClient.BatchV1().
		Jobs(NAMESPACE).Get(modelName, metav1.GetOptions{})
	require.Nil(t, err)
	require.NotNil(t, job)

	// Create a pod with same labels as job.
	pod := makePodWithLabels("fakepodmt", job.ObjectMeta.Labels)
	pod, err = k8sClient.CoreV1().Pods(NAMESPACE).Create(pod)

	// Make sure the pod was created.
	require.Nil(t, err)
	require.NotNil(t, pod)
	require.Nil(t, waitPoll(func() (bool, error) {
		pod, err = k8sClient.CoreV1().
			Pods(NAMESPACE).Get("fakepodmt", metav1.GetOptions{})
		if err == nil {
			return true, nil
		}
		return false, err
	}))

	// Update the pod container status to not ready and set restart count > 0.
	var containerStatuses []apiv1.ContainerStatus
	containerStatuses = append(containerStatuses, apiv1.ContainerStatus{
		Name:  "fakecontainername",
		Ready: false,
		State: apiv1.ContainerState{
			Terminated: &apiv1.ContainerStateTerminated{
				ExitCode: 1,
			},
		},
	})
	pod.Status.ContainerStatuses = containerStatuses
	failedPod, err := k8sClient.CoreV1().Pods(NAMESPACE).UpdateStatus(pod)

	// Make sure the pod was updated.
	require.Nil(t, err)
	require.NotNil(t, failedPod)
	require.Equal(t, failedPod.Status.ContainerStatuses[0].Ready, false)
	require.Equal(t, failedPod.Status.ContainerStatuses[0].State.Terminated.ExitCode, int32(1))

	// Check whether the GC and reconciler kicks-in:
	// - updates the job status state to Failed
	// - deletes all the sub-resources as the custom resource failed
	checkModelTrainingState(t, copy, crdClient, modelName, k8sClient, NAMESPACE, states.Failed, false)

	err = crdClient.Delete(NAMESPACE, modelName)
	require.Nil(t, err)

	modelTrainingList := &crv1.ModelTrainingList{}
	require.Nil(t, crdClient.RESTClient().
		Get().
		Resource(crv1.ModelTrainingResourcePlural).
		Do().
		Into(modelTrainingList))
	require.Equal(t, len(modelTrainingList.Items), 0)
}

func checkModelTrainingState(t *testing.T,
	modelTraining *crv1.ModelTraining,
	crdClient crd.Client,
	modelName string,
	k8sClient *kubernetes.Clientset,
	namespace string,
	state states.State,
	expectSubresourcesToExist bool) {
	// Wait for the model training crd to get created and get to an expected state.
	err := waitForModelTrainingInstanceToReachState(crdClient, NAMESPACE, modelName, state)
	require.Nil(t, err)
	checkK8sResources(t, modelTraining, k8sClient, namespace, modelName, expectSubresourcesToExist)
}

func checkK8sResources(t *testing.T, modelTraining *crv1.ModelTraining, k8sClient *kubernetes.Clientset, namespace string, modelName string, expectSubresourcesToExist bool) {
	job, err := k8sClient.BatchV1().
		Jobs(namespace).Get(modelName, metav1.GetOptions{})
	if expectSubresourcesToExist {
		require.Nil(t, err)
		require.NotNil(t, job)

		// Verify template container resource requests and limits
		jobResources := modelTraining.Spec.ResourceSpec
		depResources := job.Spec.Template.Spec.Containers[0].Resources
		expectedRequests := []string{"cpu", "memory", "alpha.kubernetes.io/nvidia-gpu"}
		expectedLimits := []string{"alpha.kubernetes.io/nvidia-gpu"}

		for _, rName := range expectedRequests {
			depQuant := depResources.Requests[apiv1.ResourceName(rName)]
			jobQuant := jobResources.Requests[rName]
			require.Equal(t, (&depQuant).MilliValue(), (&jobQuant).MilliValue())
		}
		for _, rName := range expectedLimits {
			depQuant := depResources.Limits[apiv1.ResourceName(rName)]
			jobQuant := jobResources.Requests[rName]
			require.Equal(t, (&depQuant).MilliValue(), (&jobQuant).MilliValue())
		}
	} else {
		// The job won't get deleted since we are not running the controller-manager
		// in the e2e test env.
		// However, the DELETE request to the API is posted and can be seen in the logs.
		/*require.Nil(t, waitPoll(func() (bool, error) {
			job, err = k8sClient.BatchV1().
				Jobs(namespace).Get(modelName, metav1.GetOptions{})
			if err != nil && apierrors.IsNotFound(err) {
				return true, nil
			}
			return false, err
		}))*/
	}
}

func testSpec(t *testing.T, modelTraining *crv1.ModelTraining, spec *crv1.ModelTrainingSpec) {
	// Check if all the fields are right
	require.True(t, reflect.DeepEqual(&modelTraining.Spec, spec), "Spec is not the same")
}

// WaitForModelTrainingInstanceToReachState waits for the model training job to be processed.
func waitForModelTrainingInstanceToReachState(crdClient crd.Client, namespace string, name string, state states.State) error {
	return waitPoll(func() (bool, error) {
		var modelTraining crv1.ModelTraining
		err := crdClient.RESTClient().Get().
			Resource(crv1.ModelTrainingResourcePlural).
			Namespace(namespace).
			Name(name).
			Do().Into(&modelTraining)

		if err == nil && modelTraining.Status.State == state {
			return true, nil
		}

		return false, err
	})
}

func waitPoll(waitFunc func() (bool, error)) error {
	return wait.Poll(1*time.Second, 20*time.Second, waitFunc)
}
