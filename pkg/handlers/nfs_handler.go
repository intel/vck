package handlers

import (
	"fmt"

	"github.com/golang/glog"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/client-go/kubernetes"

	kvcv1 "github.com/kubeflow/experimental-kvc/pkg/apis/kvc/v1"
	"github.com/kubeflow/experimental-kvc/pkg/resource"
)

const (
	nfsSourceType kvcv1.DataSourceType = "NFS"
)

type nfsHandler struct {
	sourceType         kvcv1.DataSourceType
	k8sClientset       kubernetes.Interface
	k8sResourceClients []resource.Client
}

// NewNFSHandler creates and returns an instance of the NFS handler.
func NewNFSHandler(k8sClientset kubernetes.Interface, resourceClients []resource.Client) DataHandler {
	return &nfsHandler{
		sourceType:         nfsSourceType,
		k8sClientset:       k8sClientset,
		k8sResourceClients: resourceClients,
	}
}

func (h *nfsHandler) GetSourceType() kvcv1.DataSourceType {
	return h.sourceType
}

func (h *nfsHandler) OnAdd(ns string, vc kvcv1.VolumeConfig, controllerRef metav1.OwnerReference) kvcv1.Volume {
	if len(vc.Labels) == 0 {
		return kvcv1.Volume{
			ID:      vc.ID,
			Message: fmt.Sprintf("labels cannot be empty"),
		}
	}

	if _, ok := vc.Options["server"]; !ok {
		return kvcv1.Volume{
			ID:      vc.ID,
			Message: fmt.Sprintf("server has to be set in options"),
		}
	}

	if _, ok := vc.Options["path"]; !ok {
		return kvcv1.Volume{
			ID:      vc.ID,
			Message: fmt.Sprintf("path has to be set in options"),
		}
	}

	if vc.AccessMode != "ReadWriteMany" && vc.AccessMode != "ReadOnlyMany" {
		return kvcv1.Volume{
			ID:      vc.ID,
			Message: fmt.Sprintf("access mode has to be either ReadWriteMany or ReadOnlyMany"),
		}
	}

	kvcName := fmt.Sprintf("%s%s", kvcNamePrefix, uuid.NewUUID())
	for _, client := range h.k8sResourceClients {
		if client.Plural() == "nodes" || client.Plural() == "pods" {
			continue
		}

		err := client.Create(ns, struct {
			kvcv1.VolumeConfig
			metav1.OwnerReference
			NS                  string
			NodeName            string
			KVCName             string
			KVCStorageClassName string
			PVType              string
			KVCOptions          map[string]string
		}{
			vc,
			controllerRef,
			ns,
			"",
			kvcName,
			"kvc",
			"nfs",
			map[string]string{
				"server": vc.Options["server"],
				"path":   vc.Options["path"],
			},
		})

		if err != nil {
			return kvcv1.Volume{
				ID:      vc.ID,
				Message: fmt.Sprintf("error during sub-resource [%s] creation: %v", client.Plural(), err),
			}
		}
	}

	return kvcv1.Volume{
		ID: vc.ID,
		VolumeSource: corev1.VolumeSource{
			PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
				ClaimName: kvcName,
			},
		},
		Message: kvcv1.SuccessfulVolumeStatusMessage,
	}
}

func (h *nfsHandler) OnDelete(ns string, vc kvcv1.VolumeConfig, controllerRef metav1.OwnerReference) {
	for _, client := range h.k8sResourceClients {
		if client.Plural() == "nodes" {
			continue
		}

		resourceList, err := client.List(ns, vc.Labels)
		if err != nil {
			glog.Warningf("[nfs-handler] OnDelete: error while listing resource [%s], %v", client.Plural(), err)
		}

		for _, resource := range resourceList {
			resControllerRef := metav1.GetControllerOf(resource)
			if resControllerRef.UID == controllerRef.UID {
				client.Delete(ns, resource.GetName())
			}
		}
	}

}
