package handlers

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	vckv1 "github.com/ppkube/vck/pkg/apis/vck/v1"
)

// DataHandler is the interface which defines the handler methods
type DataHandler interface {
	GetSourceType() vckv1.DataSourceType
	OnAdd(namespace string, vc vckv1.VolumeConfig, controllerRef metav1.OwnerReference) vckv1.Volume
	OnDelete(namespace string, vc vckv1.VolumeConfig, vStatus vckv1.Volume, controllerRef metav1.OwnerReference)
}

const (
	vckNamePrefix string = "vck-resource-"
)
