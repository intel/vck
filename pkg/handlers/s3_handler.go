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
	s3SourceType  crv1.DataSourceType = "S3"
	kvcNamePrefix string              = "kvc-resource-"
)

type s3Handler struct {
	sourceType         crv1.DataSourceType
	k8sClientset       *kubernetes.Clientset
	k8sResourceClients []resource.Client
}

func NewS3Handler(k8sClientset *kubernetes.Clientset, resourceClients []resource.Client) DataHandler {
	return &s3Handler{
		sourceType:         s3SourceType,
		k8sClientset:       k8sClientset,
		k8sResourceClients: resourceClients,
	}
}

func (h *s3Handler) GetSourceType() crv1.DataSourceType {
	return h.sourceType
}

func (h *s3Handler) OnAdd(ns string, vc crv1.VolumeConfig, controllerRef metav1.OwnerReference) crv1.VolumeClaim {
	if len(vc.Labels) == 0 {
		return crv1.VolumeClaim{
			ID:      vc.ID,
			Message: fmt.Sprintf("labels cannot be empty"),
		}
	}

	if _, ok := vc.Options["awsAccessKeyID"]; !ok {
		return crv1.VolumeClaim{
			ID:      vc.ID,
			Message: fmt.Sprintf("awsAccessKeyID key has to be set in options"),
		}
	}

	if _, ok := vc.Options["awsAccessKey"]; !ok {
		return crv1.VolumeClaim{
			ID:      vc.ID,
			Message: fmt.Sprintf("awsAccessKey key has to be set in options"),
		}
	}

	nodeClient := h.getK8SResourceClientFromPlural("nodes")
	nodeList, err := nodeClient.List(ns, map[string]string{})
	if err != nil {
		return crv1.VolumeClaim{
			ID:      vc.ID,
			Message: fmt.Sprintf("error getting node list: %v", err),
		}
	}

	// If number of nodes < replicas, then return immediately.
	if len(nodeList) < vc.Replicas {
		return crv1.VolumeClaim{
			ID: vc.ID,
			Message: fmt.Sprintf("replicas [%v] greater than number of nodes [%v]",
				vc.Replicas, len(nodeList)),
		}
	}

	nodeNames := getNodeNames(nodeList)

	kvcNames := []string{}
	for i := 0; i < vc.Replicas; i++ {
		kvcName := fmt.Sprintf("%s%s", kvcNamePrefix, uuid.NewUUID())
		kvcNames = append(kvcNames, kvcName)
		kvcDataPathSuffix := fmt.Sprintf("%s%s", kvcNamePrefix, uuid.NewUUID())
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
				KVCDataPathSuffix   string
			}{
				vc,
				controllerRef,
				ns,
				nodeNames[i],
				kvcName,
				"kvc-local",
				kvcDataPathSuffix,
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

func (h *s3Handler) OnDelete(ns string, vc crv1.VolumeConfig, controllerRef metav1.OwnerReference) {
	for _, client := range h.k8sResourceClients {
		if client.Plural() == "nodes" {
			continue
		}

		resourceList, err := client.List(ns, vc.Labels)
		if err != nil {
			glog.Warningf("[s3-handler] OnDelete: error while listing resource [%s], %v", client.Plural(), err)
		}

		for _, resource := range resourceList {
			resControllerRef := metav1.GetControllerOf(resource)
			if resControllerRef.UID == controllerRef.UID {
				client.Delete(ns, resource.GetName())
			}
		}
	}

}

func (h *s3Handler) getK8SResourceClientFromPlural(plural string) resource.Client {
	for _, client := range h.k8sResourceClients {
		if plural == client.Plural() {
			return client
		}
	}

	return nil
}

func getNodeNames(nodeList []metav1.Object) []string {
	nodeNames := []string{}

	for _, node := range nodeList {
		nodeNames = append(nodeNames, node.GetName())
	}

	return nodeNames
}
