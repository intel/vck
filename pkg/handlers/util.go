package handlers

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func getNodeNames(nodeList []metav1.Object) []string {
	nodeNames := []string{}

	for _, node := range nodeList {
		nodeNames = append(nodeNames, node.GetName())
	}

	return nodeNames
}
