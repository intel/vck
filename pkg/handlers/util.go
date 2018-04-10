package handlers

import (
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/kubeflow/experimental-kvc/pkg/resource"
)

func getNodeNames(nodeList []metav1.Object) []string {
	nodeNames := []string{}

	for _, node := range nodeList {
		nodeNames = append(nodeNames, node.GetName())
	}

	return nodeNames
}

func getK8SResourceClientFromPlural(k8sResourceClients []resource.Client, plural string) resource.Client {
	for _, client := range k8sResourceClients {
		if plural == client.Plural() {
			return client
		}
	}

	return nil
}

func waitForPodSuccess(podClient resource.Client, podName string, podNS string, timeout time.Duration) error {
	return waitPoll(func() (bool, error) {
		obj, err := podClient.Get(podNS, podName)
		if err != nil {
			return false, fmt.Errorf("error while getting pod object when checking for pod success")
		}

		pod, ok := obj.(*corev1.Pod)
		if !ok {
			return false, fmt.Errorf("object returned from podClient.Get() is not a pod")
		}

		if err == nil && pod.Status.Phase == corev1.PodSucceeded {
			return true, nil
		}

		return false, err
	}, timeout)
}

func waitPoll(waitFunc func() (bool, error), timeout time.Duration) error {
	return wait.Poll(1*time.Second, timeout, waitFunc)
}

// Returns a strategic patch for adding or removing a label for a node. Operation can be add or delete.
func patchForNodeWithLabel(oldNode *corev1.Node, label string, operation string, nodeClient resource.Client) (err error) {
	modifiedNode := oldNode.DeepCopy()

	// That label already exists on the node, so don't do anything
	if _, ok := modifiedNode.ObjectMeta.Labels[label]; !ok {
		return
	}
	switch operation {
	case "add":
		modifiedNode.ObjectMeta.Labels[label] = "true"
	case "delete":
		delete(modifiedNode.ObjectMeta.Labels, label)
	}

	oldJSON, err := json.Marshal(oldNode)
	if err != nil {
		return
	}
	modifiedJSON, err := json.Marshal(modifiedNode)
	if err != nil {
		return
	}
	patch, err := strategicpatch.CreateTwoWayMergePatch(oldJSON, modifiedJSON, corev1.Node{})
	if err != nil {
		return
	}

	glog.V(4).Infof("Original json: %v, modified json: %v", oldJSON, modifiedJSON)
	glog.V(4).Infof("Original node: %v, modified node: %v", oldNode, modifiedNode)
	if len(patch) == 0 || string(patch) == "{}" {
		return fmt.Errorf("[patchForNodeWithLabel] nothing to patch. Original: %v, Modified: %v", oldNode, modifiedNode)
	}

	nodeClient.Patch(oldNode.Name, types.StrategicMergePatchType, patch)
	return
}
