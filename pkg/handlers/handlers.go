package handlers

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kvcv1 "github.com/kubeflow/experimental-kvc/pkg/apis/kvc/v1"
)

// DataHandler is the interface which defines the handler methods
type DataHandler interface {
	GetSourceType() kvcv1.DataSourceType
	OnAdd(namespace string, vc kvcv1.VolumeConfig, controllerRef metav1.OwnerReference) kvcv1.Volume
	OnDelete(namespace string, vc kvcv1.VolumeConfig, vStatus kvcv1.Volume, controllerRef metav1.OwnerReference)
}

const (
	kvcNamePrefix string = "kvc-resource-"
)
