package handlers

import (
	"fmt"
	"github.com/golang/glog"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/client-go/kubernetes"

	"github.com/NervanaSystems/kube-controllers-go/pkg/resource"
	crv1 "github.com/NervanaSystems/kube-volume-controller/pkg/apis/cr/v1"
)

const (
	nfsSourceType crv1.DataSourceType = "NFS"
)

type nfsHandler struct {
	sourceType         crv1.DataSourceType
	k8sClientset       *kubernetes.Clientset
	k8sResourceClients []resource.Client
}

func NewNFSHandler(k8sClientset *kubernetes.Clientset, resourceClients []resource.Client) DataHandler {
	return &nfsHandler{
		sourceType:         nfsSourceType,
		k8sClientset:       k8sClientset,
		k8sResourceClients: resourceClients,
	}
}

func (h *nfsHandler) GetSourceType() crv1.DataSourceType {
	return h.sourceType
}

func (h *nfsHandler) OnAdd(ns string, vc crv1.VolumeConfig, controllerRef metav1.OwnerReference) crv1.VolumeClaim {
	if len(vc.Labels) == 0 {
		return crv1.VolumeClaim{
			ID:      vc.ID,
			Message: fmt.Sprintf("labels cannot be empty"),
		}
	}

	if _, ok := vc.Options["server"]; !ok {
		return crv1.VolumeClaim{
			ID:      vc.ID,
			Message: fmt.Sprintf("server has to be set in options"),
		}
	}

	if _, ok := vc.Options["path"]; !ok {
		return crv1.VolumeClaim{
			ID:      vc.ID,
			Message: fmt.Sprintf("path has to be set in options"),
		}
	}

	replicas := vc.Replicas

	if vc.AccessMode != "ReadWriteOnce" {
		replicas = 1
	}

	kvcNames := []string{}
	for i := 0; i < replicas; i++ {
		kvcName := fmt.Sprintf("%s%s", kvcNamePrefix, uuid.NewUUID())
		kvcNames = append(kvcNames, kvcName)
		for _, client := range h.k8sResourceClients {
			if client.Plural() == "nodes" {
				continue
			}

			err := client.Create(ns, struct {
				crv1.VolumeConfig
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
				return crv1.VolumeClaim{
					ID:      vc.ID,
					Message: fmt.Sprintf("error during sub-resource [%s] creation: %v", client.Plural(), err),
				}
			}
		}
	}

	return crv1.VolumeClaim{
		ID:       vc.ID,
		PVCNames: kvcNames,
		Message:  crv1.SuccessfulVolumeClaimMessage,
	}
}

func (h *nfsHandler) OnDelete(ns string, vc crv1.VolumeConfig, controllerRef metav1.OwnerReference) {
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
