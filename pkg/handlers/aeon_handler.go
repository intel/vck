package handlers

import (
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/client-go/kubernetes"

	"github.com/golang/glog"

	kvcv1 "github.com/kubeflow/experimental-kvc/pkg/apis/kvc/v1"
	"github.com/kubeflow/experimental-kvc/pkg/resource"
)

const (
	aeonSourceType kvcv1.DataSourceType = "Aeon"
)

type aeonHandler struct {
	sourceType         kvcv1.DataSourceType
	k8sClientset       kubernetes.Interface
	k8sResourceClients []resource.Client
}

// NewAeonHandler creates and returns an instance of the Aeon handler.
func NewAeonHandler(k8sClientset kubernetes.Interface, resourceClients []resource.Client) DataHandler {
	return &aeonHandler{
		sourceType:         aeonSourceType,
		k8sClientset:       k8sClientset,
		k8sResourceClients: resourceClients,
	}
}

func (h *aeonHandler) GetSourceType() kvcv1.DataSourceType {
	return h.sourceType
}

func (h *aeonHandler) OnAdd(ns string, vc kvcv1.VolumeConfig, controllerRef metav1.OwnerReference) kvcv1.Volume {
	if len(vc.Labels) == 0 {
		return kvcv1.Volume{
			ID:      vc.ID,
			Message: fmt.Sprintf("labels cannot be empty"),
		}
	}

	if vc.AccessMode != "ReadWriteOnce" {
		return kvcv1.Volume{
			ID:      vc.ID,
			Message: fmt.Sprintf("access mode has to be ReadWriteOnce"),
		}
	}

	if vc.Options == nil {
		vc.Options = map[string]string{}
	}

	// Check if dataPath  was set and if not set default to /var/datasets.
	if _, ok := vc.Options["dataPath"]; !ok {
		vc.Options["dataPath"] = "/var/datasets/aeon"
	}

	// In this case, we don't need to keep the timeout configurable since it's just creating a directory.
	// So defaulting it to 1m.
	timeout, err := time.ParseDuration("1m")

	nodeClient := getK8SResourceClientFromPlural(h.k8sResourceClients, "nodes")
	nodeList, err := nodeClient.List(ns, map[string]string{})
	if err != nil {
		return kvcv1.Volume{
			ID:      vc.ID,
			Message: fmt.Sprintf("error getting node list: %v", err),
		}
	}

	// If no of replicas = 0 (or not specified, return as nothing to do)
	if vc.Replicas == 0 {
		return kvcv1.Volume{
			ID:      vc.ID,
			Message: fmt.Sprintf("number of replicas is 0, no action needs to be taken"),
		}
	}
	// If number of nodes < replicas, then return immediately.
	if len(nodeList) < vc.Replicas {
		return kvcv1.Volume{
			ID: vc.ID,
			Message: fmt.Sprintf("replicas [%v] greater than number of nodes [%v]",
				vc.Replicas, len(nodeList)),
		}
	}

	nodeNames := getNodeNames(nodeList)
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
			nodeNames[i],
			kvcName,
			"kvc",
			"",
			map[string]string{
				"path": fmt.Sprintf("%s/%s", vc.Options["dataPath"], kvcDataPathSuffix),
			},
		})

		if err != nil {
			return kvcv1.Volume{
				ID:      vc.ID,
				Message: fmt.Sprintf("error during sub-resource [%s] creation: %v", podClient.Plural(), err),
			}
		}
	}

	for _, kvcName := range kvcNames {
		err := waitForPodSuccess(podClient, kvcName, ns, timeout)
		if err != nil {
			return kvcv1.Volume{
				ID:      vc.ID,
				Message: fmt.Sprintf("error during cache creation using pod [name: %v]: %v", kvcName, err),
			}
		}
	}

	return kvcv1.Volume{
		ID: vc.ID,
		VolumeSource: corev1.VolumeSource{
			HostPath: &corev1.HostPathVolumeSource{
				Path: fmt.Sprintf("%s/%s", vc.Options["dataPath"], kvcDataPathSuffix),
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
		Message: kvcv1.SuccessfulVolumeStatusMessage,
	}
}

func (h *aeonHandler) OnDelete(ns string, vc kvcv1.VolumeConfig, controllerRef metav1.OwnerReference) {
	podClient := getK8SResourceClientFromPlural(h.k8sResourceClients, "pods")
	podList, err := podClient.List(ns, vc.Labels)
	if err != nil {
		glog.Warningf("[aeon-handler] OnDelete: error while listing resource [%s], %v", podClient.Plural(), err)
	}

	for _, resource := range podList {
		resControllerRef := metav1.GetControllerOf(resource)
		if resControllerRef.UID == controllerRef.UID {
			podClient.Delete(ns, resource.GetName())
		}
	}

}
