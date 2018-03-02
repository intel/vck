package handlers

import (
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/NervanaSystems/kube-volume-controller/pkg/resource"
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
