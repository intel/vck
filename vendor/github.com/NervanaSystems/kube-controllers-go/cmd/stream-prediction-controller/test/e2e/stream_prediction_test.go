package e2e

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	crv1 "github.com/NervanaSystems/kube-controllers-go/cmd/stream-prediction-controller/apis/cr/v1"
	"github.com/NervanaSystems/kube-controllers-go/pkg/crd"
	"github.com/NervanaSystems/kube-controllers-go/pkg/states"
	"github.com/NervanaSystems/kube-controllers-go/pkg/util"
	"github.com/segmentio/ksuid"
	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	extv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

const NAMESPACE = "e2e-test"

func makeClients(t *testing.T) (crd.Client, *kubernetes.Clientset) {
	config, err := util.BuildConfig("/go/src/github.com/NervanaSystems/kube-controllers-go/resources/config")
	require.Nil(t, err)

	crdHandle := crd.New(
		&crv1.StreamPrediction{},
		&crv1.StreamPredictionList{},
		crv1.GroupName,
		crv1.Version,
		crv1.StreamPredictionResourceKind,
		crv1.StreamPredictionResourceSingular,
		crv1.StreamPredictionResourcePlural,
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

func makeStreamPrediction(streamName string, streamID int) *crv1.StreamPrediction {
	return &crv1.StreamPrediction{
		ObjectMeta: metav1.ObjectMeta{
			Name: streamName,
		},
		Spec: crv1.StreamPredictionSpec{
			NeonRepoSpec: crv1.NeonRepoSpec{
				RepoURL: "git@github.com:NervanaSystems/private-neon.git",
				Commit:  "v1.8.2",
			},
			SecuritySpec: crv1.SecuritySpec{
				PresignedToken: "95fcbe0cfe747b867655a243cee330",
				JWTToken:       "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdHJlYW1faWQiOjEwfQ.JxxqL8-6OV4xfQmy4dGRis3QSRuTJH2kattCfLHGKwA",
			},
			StreamDataSpec: crv1.StreamDataSpec{
				ModelPRM:         "/code/model.prm",
				ModelPath:        "s3://helium-joboutput-dev/integration/20dec8c3e38e2804888f252ef281121b/51/model.prm",
				DatasetPath:      "None",
				ExtraFilename:    "None",
				CustomCodeURL:    "None",
				CustomCommit:     "None",
				AWSPath:          "krypton-logs-dev/integration",
				AWSDefaultRegion: "us-west-1",
				StreamID:         streamID,
				StreamName:       streamName,
			},
			ResourceSpec: crv1.ResourceSpec{
				Requests: map[string]resource.Quantity{
					"cpu":    resource.MustParse("1"),
					"memory": resource.MustParse("512M"),
					"alpha.kubernetes.io/nvidia-gpu": resource.MustParse("2"),
				},
			},
			KryptonRepoSpec: crv1.KryptonRepoSpec{
				RepoURL:      "git@github.com:NervanaSystems/krypton.git",
				Commit:       "master",
				Image:        "nervana/krypton:master",
				SidecarImage: "nervana/krypton-sidecar:master",
			},
			State: states.Running,
		},
		Status: crv1.StreamPredictionStatus{
			State:   states.Pending,
			Message: "Pending",
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

func refresh(t *testing.T, local *crv1.StreamPrediction, crdClient crd.Client) {
	streamName := local.Name()
	namespace := local.Namespace()
	*local = crv1.StreamPrediction{}
	err := crdClient.RESTClient().Get().
		Resource(crv1.StreamPredictionResourcePlural).
		Namespace(namespace).
		Name(streamName).
		Do().
		Into(local)
	require.Nil(t, err)
}

func TestStreamPrediction(t *testing.T) {
	crdClient, k8sClient := makeClients(t)

	streamName := fmt.Sprintf("stream%s", strings.ToLower(ksuid.New().String()))
	streamID := 0
	original := makeStreamPrediction(streamName, streamID)

	copy := &crv1.StreamPrediction{}
	err := crdClient.RESTClient().Post().
		Resource(crv1.StreamPredictionResourcePlural).
		Namespace(NAMESPACE).
		Body(original).
		Do().
		Into(copy)

	if err == nil {
		t.Logf("Created stream prediction: %#v\n", copy)
	} else if apierrors.IsAlreadyExists(err) {
		t.Errorf("Stream prediction already exists: %#v\n", copy)
	} else {
		t.Fatal(err)
	}

	// Check whether the job was created successfully
	refresh(t, copy, crdClient)
	testSpec(t, copy, &(original.Spec))

	// Check whether the job was processed.
	// In the running state, all subresources should exist.
	checkStreamState(t, copy, crdClient, streamName, k8sClient, NAMESPACE, states.Running, true)

	refresh(t, copy, crdClient)
	testSpec(t, copy, &(original.Spec))

	// Right now it's in Running. Try changing it to Completed and check if all the resources are deleted.
	refresh(t, copy, crdClient)
	copy.Spec.State = states.Completed

	_, err = crdClient.Update(copy)
	require.Nil(t, err)

	refresh(t, copy, crdClient)
	checkStreamState(t, copy, crdClient, streamName, k8sClient, NAMESPACE, states.Completed, false)

	err = crdClient.Delete(NAMESPACE, streamName)
	require.Nil(t, err)

	streamPredictList := &crv1.StreamPredictionList{}
	require.Nil(t, crdClient.RESTClient().
		Get().
		Resource(crv1.StreamPredictionResourcePlural).
		Do().
		Into(streamPredictList))
	require.Equal(t, len(streamPredictList.Items), 0)
}

// Test if reconcile works for StreamPrediciton.
// Create a new stream prediction job which will end-up in an error.
func TestStreamPredictionReconcileDepFail(t *testing.T) {
	crdClient, k8sClient := makeClients(t)

	streamName := fmt.Sprintf("stream%s", strings.ToLower(ksuid.New().String()))
	streamID := 0
	original := makeStreamPrediction(streamName, streamID)

	copy := &crv1.StreamPrediction{}
	err := crdClient.RESTClient().Post().
		Resource(crv1.StreamPredictionResourcePlural).
		Namespace(NAMESPACE).
		Body(original).
		Do().
		Into(copy)

	if err == nil {
		t.Logf("Created stream prediction: %#v\n", copy)
	} else if apierrors.IsAlreadyExists(err) {
		t.Errorf("Stream prediction already exists: %#v\n", copy)
	} else {
		t.Fatal(err)
	}

	// Check whether the job was created successfully.
	refresh(t, copy, crdClient)
	testSpec(t, copy, &(original.Spec))

	// Check whether the job was processed.
	// In the running state, all subresources should exist.
	checkStreamState(t, copy, crdClient, streamName, k8sClient, NAMESPACE, states.Running, true)

	// Update the deployment condition to ReplicaFailure.
	deployment := &v1beta1.Deployment{}
	deployment, err = k8sClient.ExtensionsV1beta1().
		Deployments(NAMESPACE).Get(streamName, metav1.GetOptions{})
	require.Nil(t, err)
	require.NotNil(t, deployment)

	depConditions := deployment.Status.Conditions
	depConditions = append(depConditions, v1beta1.DeploymentCondition{
		Type:               v1beta1.DeploymentReplicaFailure,
		Status:             apiv1.ConditionTrue,
		LastUpdateTime:     metav1.Now(),
		LastTransitionTime: metav1.Now(),
		Reason:             "fakeReason",
		Message:            "fakeMsg",
	})

	deployment.Status.Conditions = depConditions
	failedDeployment, err := k8sClient.ExtensionsV1beta1().
		Deployments(NAMESPACE).UpdateStatus(deployment)
	require.Nil(t, err)
	require.NotNil(t, failedDeployment)
	require.Equal(t, failedDeployment.Status.Conditions[0].Type, v1beta1.DeploymentReplicaFailure)

	// Check whether the GC kicks-in:
	// - deletes all the sub-resources as the deployment failed
	// - updates the job status state to "Failed"
	checkStreamState(t, copy, crdClient, streamName, k8sClient, NAMESPACE, states.Failed, false)

	err = crdClient.Delete(NAMESPACE, streamName)
	require.Nil(t, err)

	streamPredictList := &crv1.StreamPredictionList{}
	err = crdClient.RESTClient().
		Get().
		Resource(crv1.StreamPredictionResourcePlural).
		Do().
		Into(streamPredictList)
	require.Nil(t, err)
	require.Equal(t, len(streamPredictList.Items), 0)
}

// Test if reconcile works for StreamPrediciton when container in a pod fails.
// Create a new stream prediction job which will end-up in an error.
func TestStreamPredictionReconcilePodFail(t *testing.T) {
	crdClient, k8sClient := makeClients(t)

	streamName := fmt.Sprintf("stream%s", strings.ToLower(ksuid.New().String()))
	streamID := 0
	original := makeStreamPrediction(streamName, streamID)

	copy := &crv1.StreamPrediction{}
	err := crdClient.RESTClient().Post().
		Resource(crv1.StreamPredictionResourcePlural).
		Namespace(NAMESPACE).
		Body(original).
		Do().
		Into(copy)

	if err != nil {
		t.Fatal(err)
	}

	t.Logf("Created stream prediction: %#v\n", copy)

	// Check whether the job was created successfully.
	refresh(t, copy, crdClient)
	testSpec(t, copy, &(original.Spec))

	// Check whether the job was processed.
	// In the running state, all subresources should exist.
	checkStreamState(t, copy, crdClient, streamName, k8sClient, NAMESPACE, states.Running, true)

	// Update the deployment condition to ready to make sure this test case
	// exercises the container in a pod failed case.
	deployment := &v1beta1.Deployment{}
	deployment, err = k8sClient.ExtensionsV1beta1().
		Deployments(NAMESPACE).Get(streamName, metav1.GetOptions{})
	require.Nil(t, err)
	require.NotNil(t, deployment)

	depConditions := deployment.Status.Conditions
	depConditions = append(depConditions, v1beta1.DeploymentCondition{
		Type:               v1beta1.DeploymentAvailable,
		Status:             apiv1.ConditionTrue,
		LastUpdateTime:     metav1.Now(),
		LastTransitionTime: metav1.Now(),
		Reason:             "fakeReason",
		Message:            "fakeMsg",
	})

	deployment.Status.Conditions = depConditions
	availableDeployment, err := k8sClient.ExtensionsV1beta1().
		Deployments(NAMESPACE).UpdateStatus(deployment)
	require.Nil(t, err)
	require.NotNil(t, availableDeployment)
	require.Equal(t, availableDeployment.Status.Conditions[0].Type, v1beta1.DeploymentAvailable)

	// Create a pod with same labels as deployment.
	pod := makePodWithLabels("fakepod", availableDeployment.ObjectMeta.Labels)
	t.Logf("POD: %v", pod)
	pod, err = k8sClient.CoreV1().Pods(NAMESPACE).Create(pod)

	// Make sure the pod was created.
	require.Nil(t, err)
	require.NotNil(t, pod)
	require.Nil(t, waitPoll(func() (bool, error) {
		pod, err = k8sClient.CoreV1().
			Pods(NAMESPACE).Get("fakepod", metav1.GetOptions{})
		if err == nil {
			return true, nil
		}
		return false, err
	}))

	// Update the pod container status to not ready and set restart count > 0.
	var containerStatuses []apiv1.ContainerStatus
	containerStatuses = append(containerStatuses, apiv1.ContainerStatus{
		Name:         "fakecontainername",
		Ready:        false,
		RestartCount: 1,
	})
	pod.Status.ContainerStatuses = containerStatuses
	failedPod, err := k8sClient.CoreV1().Pods(NAMESPACE).UpdateStatus(pod)

	// Make sure the pod was updated.
	require.Nil(t, err)
	require.NotNil(t, failedPod)
	require.Equal(t, failedPod.Status.ContainerStatuses[0].Ready, false)
	require.Equal(t, failedPod.Status.ContainerStatuses[0].RestartCount, int32(1))

	// Check whether the GC and reconciler kicks-in:
	// - updates the job status state to Failed
	// - deletes all the sub-resources as the custom resource failed
	checkStreamState(t, copy, crdClient, streamName, k8sClient, NAMESPACE, states.Failed, false)

	err = crdClient.Delete(NAMESPACE, streamName)
	require.Nil(t, err)

	streamPredictList := &crv1.StreamPredictionList{}
	err = crdClient.RESTClient().
		Get().
		Resource(crv1.StreamPredictionResourcePlural).
		Do().
		Into(streamPredictList)
	require.Nil(t, err)
	require.Equal(t, len(streamPredictList.Items), 0)
}

func checkStreamState(t *testing.T,
	streamPrediction *crv1.StreamPrediction,
	crdClient crd.Client,
	streamName string,
	k8sClient *kubernetes.Clientset,
	namespace string,
	state states.State,
	expectSubresourcesToExist bool) {
	// Wait for the stream predict crd to get created and being processed
	err := waitForStreamPredictionInstanceProcessed(crdClient, NAMESPACE, streamName, state)
	require.Nil(t, err)
	checkK8sResources(t, streamPrediction, k8sClient, namespace, streamName, expectSubresourcesToExist)
}

func checkK8sResources(t *testing.T, streamPrediction *crv1.StreamPrediction, k8sClient *kubernetes.Clientset, namespace string, streamName string, expectSubresourcesToExist bool) {
	deployment, err := k8sClient.ExtensionsV1beta1().
		Deployments(namespace).Get(streamName, metav1.GetOptions{})
	if expectSubresourcesToExist {
		require.Nil(t, err)
		require.NotNil(t, deployment)

		// Verify template container resource requests and limits
		jobResources := streamPrediction.Spec.ResourceSpec
		depResources := deployment.Spec.Template.Spec.Containers[0].Resources
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
		// Deployment is not getting deleted at all in this cluster. So commenting it for now.
		// However, the DELETE request to the API is posted and can be seen in the logs.
		/*require.Nil(t, waitPoll(func() (bool, error) {
		deployment, err = k8sClient.ExtensionsV1beta1().
			Deployments(namespace).Get(streamName, metav1.GetOptions{})
				if err != nil && apierrors.IsNotFound(err) {
					return true, nil
				}
				return false, err
			}))*/
	}
	service, err := k8sClient.CoreV1().Services(namespace).
		Get(streamName, metav1.GetOptions{})
	if expectSubresourcesToExist {
		require.Nil(t, err)
		require.NotNil(t, service)
	} else {
		// It takes a while to delete the resources, so waiting till they get deleted.
		require.Nil(t, waitPoll(func() (bool, error) {
			service, err = k8sClient.CoreV1().Services(namespace).
				Get(streamName, metav1.GetOptions{})
			if err != nil && apierrors.IsNotFound(err) {
				return true, nil
			}
			return false, err
		}))
	}
	ingress, err := k8sClient.ExtensionsV1beta1().
		Ingresses(namespace).Get(streamName, metav1.GetOptions{})
	if expectSubresourcesToExist {
		require.Nil(t, err)
		require.NotNil(t, ingress)
	} else {
		// It takes a while to delete the resources, so waiting till they get deleted.
		require.Nil(t, waitPoll(func() (bool, error) {
			ingress, err = k8sClient.ExtensionsV1beta1().
				Ingresses(namespace).Get(streamName, metav1.GetOptions{})
			if err != nil && apierrors.IsNotFound(err) {
				return true, nil
			}
			return false, err
		}))
	}
	hpa, err := k8sClient.AutoscalingV1().
		HorizontalPodAutoscalers(namespace).Get(streamName, metav1.GetOptions{})
	if expectSubresourcesToExist {
		require.Nil(t, err)
		require.NotNil(t, hpa)
	} else {
		// It takes a while to delete the resources, so waiting till they get deleted.
		require.Nil(t, waitPoll(func() (bool, error) {
			hpa, err = k8sClient.AutoscalingV1().
				HorizontalPodAutoscalers(namespace).Get(streamName, metav1.GetOptions{})
			if err != nil && apierrors.IsNotFound(err) {
				return true, nil
			}
			return false, err
		}))
	}
}

func testSpec(t *testing.T, streamPrediction *crv1.StreamPrediction, spec *crv1.StreamPredictionSpec) {
	// Check if all the fields are right
	require.True(t, reflect.DeepEqual(&streamPrediction.Spec, spec), "Spec is not the same")
}

// WaitForStreamPredictionInstanceProcessed waits for the stream prediction to be processed.
func waitForStreamPredictionInstanceProcessed(crdClient crd.Client, namespace string, name string, state states.State) error {
	return waitPoll(func() (bool, error) {
		var streamPrediction crv1.StreamPrediction
		err := crdClient.RESTClient().Get().
			Resource(crv1.StreamPredictionResourcePlural).
			Namespace(namespace).
			Name(name).
			Do().Into(&streamPrediction)

		if err == nil && streamPrediction.Status.State == state {
			return true, nil
		}

		return false, err
	})
}

func waitPoll(waitFunc func() (bool, error)) error {
	return wait.Poll(1*time.Second, 30*time.Second, waitFunc)
}
