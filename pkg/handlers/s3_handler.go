package handlers

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/client-go/kubernetes"

	"github.com/golang/glog"

	"github.com/NervanaSystems/kube-controllers-go/pkg/resource"
	crv1 "github.com/NervanaSystems/kube-volume-controller/pkg/apis/cr/v1"
)

const (
	s3SourceType crv1.DataSourceType = "S3"
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

func (h *s3Handler) OnAdd(ns string, vc crv1.VolumeConfig, controllerRef metav1.OwnerReference) crv1.Volume {
	if len(vc.Labels) == 0 {
		return crv1.Volume{
			ID:      vc.ID,
			Message: fmt.Sprintf("labels cannot be empty"),
		}
	}

	if _, ok := vc.Options["awsCredentialsSecretName"]; !ok {
		return crv1.Volume{
			ID:      vc.ID,
			Message: fmt.Sprintf("awsCredentialsSecretName key has to be set in options"),
		}
	}

	nodeClient := h.getK8SResourceClientFromPlural("nodes")
	nodeList, err := nodeClient.List(ns, map[string]string{})
	if err != nil {
		return crv1.Volume{
			ID:      vc.ID,
			Message: fmt.Sprintf("error getting node list: %v", err),
		}
	}

	// If number of nodes < replicas, then return immediately.
	if len(nodeList) < vc.Replicas {
		return crv1.Volume{
			ID: vc.ID,
			Message: fmt.Sprintf("replicas [%v] greater than number of nodes [%v]",
				vc.Replicas, len(nodeList)),
		}
	}

	nodeNames := getNodeNames(nodeList)
	recursiveFlag := ""
	if strings.HasSuffix(vc.SourceURL, "/") {
		recursiveFlag = "--recursive"
	}

	usedNodeNames := []string{}
	podClient := h.getK8SResourceClientFromPlural("pods")
	kvcDataPathSuffix := fmt.Sprintf("%s%s", kvcNamePrefix, uuid.NewUUID())
	for i := 0; i < vc.Replicas; i++ {
		kvcName := fmt.Sprintf("%s%s", kvcNamePrefix, uuid.NewUUID())
		usedNodeNames = append(usedNodeNames, nodeNames[i])

		err := podClient.Create(ns, struct {
			crv1.VolumeConfig
			metav1.OwnerReference
			NS                  string
			NodeName            string
			KVCName             string
			KVCStorageClassName string
			PVType              string
			RecursiveOption     string
			KVCOptions          map[string]string
		}{
			vc,
			controllerRef,
			ns,
			nodeNames[i],
			kvcName,
			"kvc",
			"",
			recursiveFlag,
			map[string]string{
				"path": fmt.Sprintf("/var/datasets/%s", kvcDataPathSuffix),
			},
		})

		if err != nil {
			return crv1.Volume{
				ID:      vc.ID,
				Message: fmt.Sprintf("error during sub-resource [%s] creation: %v", podClient.Plural(), err),
			}
		}
	}

	return crv1.Volume{
		ID: vc.ID,
		VolumeSource: corev1.VolumeSource{
			HostPath: &corev1.HostPathVolumeSource{
				Path: fmt.Sprintf("/var/datasets/%s", kvcDataPathSuffix),
			},
		},
		NodeAffinity: corev1.NodeAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
				NodeSelectorTerms: []corev1.NodeSelectorTerm{
					{
						MatchExpressions: []corev1.NodeSelectorRequirement{
							{
								Key:      "kubernetes.io/hostname",
								Operator: corev1.NodeSelectorOpIn,
								Values:   usedNodeNames,
							},
						},
					},
				},
			},
		},
		Message: crv1.SuccessfulVolumeStatusMessage,
	}
}

func (h *s3Handler) OnDelete(ns string, vc crv1.VolumeConfig, controllerRef metav1.OwnerReference) {

	podClient := h.getK8SResourceClientFromPlural("pods")
	podList, err := podClient.List(ns, vc.Labels)
	if err != nil {
		glog.Warningf("[s3-handler] OnDelete: error while listing resource [%s], %v", podClient.Plural(), err)
	}

	for _, resource := range podList {
		resControllerRef := metav1.GetControllerOf(resource)
		if resControllerRef.UID == controllerRef.UID {
			podClient.Delete(ns, resource.GetName())
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
