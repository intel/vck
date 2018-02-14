package handlers

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	crv1 "github.com/NervanaSystems/kube-volume-controller/pkg/apis/cr/v1"
)

type DataHandler interface {
	GetSourceType() crv1.DataSourceType
	OnAdd(namespace string, vc crv1.VolumeConfig, controllerRef metav1.OwnerReference) crv1.Volume
	OnDelete(namespace string, vc crv1.VolumeConfig, controllerRef metav1.OwnerReference)
}

const (
	kvcNamePrefix string = "kvc-resource-"
)
