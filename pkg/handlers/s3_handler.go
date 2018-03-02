package handlers

import (
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/client-go/kubernetes"

	"github.com/golang/glog"

	crv1 "github.com/NervanaSystems/kube-volume-controller/pkg/apis/cr/v1"
	"github.com/NervanaSystems/kube-volume-controller/pkg/resource"
)

const (
	s3SourceType crv1.DataSourceType = "S3"
)

type s3Handler struct {
	sourceType         crv1.DataSourceType
	k8sClientset       *kubernetes.Clientset
	k8sResourceClients []resource.Client
}

// NewS3Handler creates and returns an instance of the NFS handler.
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

	if vc.AccessMode != "ReadWriteOnce" {
		return crv1.Volume{
			ID:      vc.ID,
			Message: fmt.Sprintf("access mode has to be ReadWriteOnce"),
		}
	}

	// Set the default timeout for data download using a pod to 5 minutes.
	timeout, err := time.ParseDuration("5m")
	// Check if timeout for data download was set and use it.
	if _, ok := vc.Options["timeoutForDataDownload"]; ok {
		timeout, err = time.ParseDuration(vc.Options["timeoutForDataDownload"])
		if err != nil {
			return crv1.Volume{
				ID:      vc.ID,
				Message: fmt.Sprintf("error while parsing timeout for data download: %v", err),
			}
		}
	}

	nodeClient := getK8SResourceClientFromPlural(h.k8sResourceClients, "nodes")
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
	kvcNames := []string{}
	podClient := getK8SResourceClientFromPlural(h.k8sResourceClients, "pods")
	kvcDataPathSuffix := fmt.Sprintf("%s%s", kvcNamePrefix, uuid.NewUUID())
	for i := 0; i < vc.Replicas; i++ {
		kvcName := fmt.Sprintf("%s%s", kvcNamePrefix, uuid.NewUUID())
		kvcNames = append(kvcNames, kvcName)

		// Collect node names for providing node affinity details.
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

	for _, kvcName := range kvcNames {
		err := waitForPodSuccess(podClient, kvcName, ns, timeout)
		if err != nil {
			return crv1.Volume{
				ID:      vc.ID,
				Message: fmt.Sprintf("error during data download using pod [name: %v]: %v", kvcName, err),
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
	podClient := getK8SResourceClientFromPlural(h.k8sResourceClients, "pods")
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
