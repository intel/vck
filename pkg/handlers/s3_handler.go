package handlers

import (
	"fmt"
	"net/url"
	"strings"
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
	s3SourceType kvcv1.DataSourceType = "S3"
)

type s3Handler struct {
	sourceType         kvcv1.DataSourceType
	k8sClientset       kubernetes.Interface
	k8sResourceClients []resource.Client
}

// NewS3Handler creates and returns an instance of the NFS handler.
func NewS3Handler(k8sClientset kubernetes.Interface, resourceClients []resource.Client) DataHandler {
	return &s3Handler{
		sourceType:         s3SourceType,
		k8sClientset:       k8sClientset,
		k8sResourceClients: resourceClients,
	}
}

func (h *s3Handler) GetSourceType() kvcv1.DataSourceType {
	return h.sourceType
}

func (h *s3Handler) OnAdd(ns string, vc kvcv1.VolumeConfig, controllerRef metav1.OwnerReference) kvcv1.Volume {
	if len(vc.Labels) == 0 {
		return kvcv1.Volume{
			ID:      vc.ID,
			Message: fmt.Sprintf("labels cannot be empty"),
		}
	}

	if _, ok := vc.Options["awsCredentialsSecretName"]; !ok {
		return kvcv1.Volume{
			ID:      vc.ID,
			Message: fmt.Sprintf("awsCredentialsSecretName key has to be set in options"),
		}
	}

	if vc.AccessMode != "ReadWriteOnce" {
		return kvcv1.Volume{
			ID:      vc.ID,
			Message: fmt.Sprintf("access mode has to be ReadWriteOnce"),
		}
	}

	// Check if dataPath  was set and  if not set default to /var/datasets.
	if _, ok := vc.Options["dataPath"]; !ok {
		vc.Options["dataPath"] = "/var/datasets"
	}

	// Set the default timeout for data download using a pod to 5 minutes.
	timeout, err := time.ParseDuration("5m")
	// Check if timeout for data download was set and use it.
	if _, ok := vc.Options["timeoutForDataDownload"]; ok {
		timeout, err = time.ParseDuration(vc.Options["timeoutForDataDownload"])
		if err != nil {
			return kvcv1.Volume{
				ID:      vc.ID,
				Message: fmt.Sprintf("error while parsing timeout for data download: %v", err),
			}
		}
	}

	nodeClient := getK8SResourceClientFromPlural(h.k8sResourceClients, "nodes")
	nodeList, err := nodeClient.List(ns, map[string]string{})
	if err != nil {
		return kvcv1.Volume{
			ID:      vc.ID,
			Message: fmt.Sprintf("error getting node list: %v", err),
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

	recursiveFlag := ""
	if strings.HasSuffix(vc.SourceURL, "/") {
		recursiveFlag = "--recursive"
	}

	s3URL, err := url.Parse(vc.SourceURL)
	if err != nil {
		return kvcv1.Volume{
			ID:      vc.ID,
			Message: fmt.Sprintf("error while parsing URL [%s]: %v", vc.SourceURL, err),
		}
	}
	bucketName := s3URL.Host
	bucketPath := s3URL.Path

	kvcNames := []string{}
	podClient := getK8SResourceClientFromPlural(h.k8sResourceClients, "pods")
	kvcDataPathSuffix := fmt.Sprintf("%s%s", kvcNamePrefix, uuid.NewUUID())
	for i := 0; i < vc.Replicas; i++ {
		kvcName := fmt.Sprintf("%s%s", kvcNamePrefix, uuid.NewUUID())
		kvcNames = append(kvcNames, kvcName)

		// patch nodes with the correct label
		node, err := nodeClient.Get("", nodeNames[i])
		if err != nil {
			return kvcv1.Volume{
				ID:      vc.ID,
				Message: fmt.Sprintf("could not get node %s, error: %v", nodeNames[i], err),
			}
		}
		err = patchForNodeWithLabels(node.(*corev1.Node), []string{fmt.Sprintf("%s/%s", kvcv1.GroupName, vc.ID)}, "add", nodeClient)

		if err != nil {
			return kvcv1.Volume{
				ID:      vc.ID,
				Message: fmt.Sprintf("could not label node %s, error: %v", nodeNames[i], err),
			}
		}

		err = podClient.Create(ns, struct {
			kvcv1.VolumeConfig
			metav1.OwnerReference
			NS                  string
			KVCName             string
			KVCStorageClassName string
			PVType              string
			RecursiveOption     string
			BucketName          string
			BucketPath          string
			KVCOptions          map[string]string
		}{
			vc,
			controllerRef,
			ns,
			kvcName,
			"kvc",
			"",
			recursiveFlag,
			bucketName,
			bucketPath,
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

	usedNodeNames := []string{}
	for _, kvcName := range kvcNames {
		err := waitForPodSuccess(podClient, kvcName, ns, timeout)
		if err != nil {
			return kvcv1.Volume{
				ID: vc.ID,
				// TODO(balajismaniam): append pod logs to this message if possible.
				Message: fmt.Sprintf("error during data download using pod [name: %v]: %v", kvcName, err),
			}
		}

		podObj, err := podClient.Get(ns, kvcName)
		if err != nil {
			return kvcv1.Volume{
				ID:      vc.ID,
				Message: fmt.Sprintf("error getting pod [name: %v]: %v", kvcName, err),
			}
		}

		pod, ok := podObj.(*corev1.Pod)
		if !ok {
			return kvcv1.Volume{
				ID:      vc.ID,
				Message: fmt.Sprintf("object returned from podclient.Get() is not a pod"),
			}
		}

		usedNodeNames = append(usedNodeNames, pod.Spec.NodeName)
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
								Key:      fmt.Sprintf("%s/%s", kvcv1.GroupName, vc.ID),
								Operator: corev1.NodeSelectorOpExists,
							},
						},
					},
				},
			},
		},
		Message: kvcv1.SuccessfulVolumeStatusMessage,
	}
}

func (h *s3Handler) OnDelete(ns string, vc kvcv1.VolumeConfig, controllerRef metav1.OwnerReference) {
	podClient := getK8SResourceClientFromPlural(h.k8sResourceClients, "pods")
	podList, err := podClient.List(ns, vc.Labels)
	if err != nil {
		glog.Warningf("[s3-handler] OnDelete: error while listing resource [%s], %v", podClient.Plural(), err)
	}

	for _, resource := range podList {
		resControllerRef := metav1.GetControllerOf(resource)
		if resControllerRef == nil {
			continue
		}

		if resControllerRef.UID == controllerRef.UID {
			podClient.Delete(ns, resource.GetName())
		}
	}

	// Delete the annotation for the node
	// Get the node list based on the vc.ID
	nodeClient := getK8SResourceClientFromPlural(h.k8sResourceClients, "nodes")

	nodeList, err := nodeClient.List("", map[string]string{fmt.Sprintf("%s/%s", kvcv1.GroupName, vc.ID): "true"})
	if err != nil {
		glog.Warningf("[s3-handler] OnDelete: error while listing nodes %v", err)
		return
	}
	nodeNames := getNodeNames(nodeList)

	for _, nodeName := range nodeNames {

		node, err := nodeClient.Get("", nodeName)
		if err != nil {
			glog.Warningf("[s3-handler] OnDelete: error while getting node: %v", err)
		}

		err = patchForNodeWithLabels(node.(*corev1.Node), []string{fmt.Sprintf("%s/%s", kvcv1.GroupName, vc.ID)}, "delete", nodeClient)
		if err != nil {
			glog.Warningf("[s3-handler] OnDelete: error while deleting label for node nodes %v", err)
		}
	}
}
