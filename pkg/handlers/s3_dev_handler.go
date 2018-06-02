package handlers

import (
	"fmt"
	"math/rand"
	"net/url"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/client-go/kubernetes"

	"github.com/golang/glog"

	vckv1 "github.com/IntelAI/vck/pkg/apis/vck/v1"
	"github.com/IntelAI/vck/pkg/resource"
)

const (
	s3DevSourceType vckv1.DataSourceType = "S3-Dev"
)

type s3DevHandler struct {
	sourceType         vckv1.DataSourceType
	k8sClientset       kubernetes.Interface
	k8sResourceClients []resource.Client
}

// NewS3DevHandler creates and returns an instance of the NFS handler.
func NewS3DevHandler(k8sClientset kubernetes.Interface, resourceClients []resource.Client) DataHandler {
	return &s3DevHandler{
		sourceType:         s3DevSourceType,
		k8sClientset:       k8sClientset,
		k8sResourceClients: resourceClients,
	}
}

func (h *s3DevHandler) GetSourceType() vckv1.DataSourceType {
	return h.sourceType
}

func (h *s3DevHandler) OnAdd(ns string, vc vckv1.VolumeConfig, controllerRef metav1.OwnerReference) vckv1.Volume {
	if len(vc.Labels) == 0 {
		return vckv1.Volume{
			ID:      vc.ID,
			Message: fmt.Sprintf("labels cannot be empty"),
		}
	}

	if _, ok := vc.Options["awsCredentialsSecretName"]; !ok {
		return vckv1.Volume{
			ID:      vc.ID,
			Message: fmt.Sprintf("awsCredentialsSecretName key has to be set in options"),
		}
	}

	if vc.AccessMode != "ReadWriteOnce" {
		return vckv1.Volume{
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
			return vckv1.Volume{
				ID:      vc.ID,
				Message: fmt.Sprintf("error while parsing timeout for data download: %v", err),
			}
		}
	}

	nodeClient := getK8SResourceClientFromPlural(h.k8sResourceClients, "nodes")
	nodeList, err := nodeClient.List(ns, map[string]string{})
	if err != nil {
		return vckv1.Volume{
			ID:      vc.ID,
			Message: fmt.Sprintf("error getting node list: %v", err),
		}
	}

	// If number of nodes < replicas, then return immediately.
	if len(nodeList) < vc.Replicas {
		return vckv1.Volume{
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

	s3URL, err := url.Parse(vc.SourceURL)
	if err != nil {
		return vckv1.Volume{
			ID:      vc.ID,
			Message: fmt.Sprintf("error while parsing URL [%s]: %v", vc.SourceURL, err),
		}
	}
	bucketName := s3URL.Host
	bucketPath := s3URL.Path

	vckName := fmt.Sprintf("%s%s", vckNamePrefix, uuid.NewUUID())
	vckDataPathSuffix := fmt.Sprintf("%s%s", vckNamePrefix, uuid.NewUUID())
	rand.Seed(time.Now().Unix())
	for _, client := range h.k8sResourceClients {
		if client.Plural() == "nodes" {
			continue
		}

		err := client.Create(ns, struct {
			vckv1.VolumeConfig
			metav1.OwnerReference
			NS                  string
			NodeName            string
			VCKName             string
			VCKStorageClassName string
			PVType              string
			VCKOp               string
			RecursiveOption     string
			BucketName          string
			BucketPath          string
			VCKOptions          map[string]string
		}{
			vc,
			controllerRef,
			ns,
			nodeNames[rand.Intn(len(nodeNames))],
			vckName,
			"vck",
			"local",
			"add",
			recursiveFlag,
			bucketName,
			bucketPath,
			map[string]string{
				"path": fmt.Sprintf("%s/%s", vc.Options["dataPath"], vckDataPathSuffix),
			},
		})

		if err != nil {
			return vckv1.Volume{
				ID:      vc.ID,
				Message: fmt.Sprintf("error during sub-resource [%s] creation: %v", client.Plural(), err),
			}
		}
	}

	podClient := getK8SResourceClientFromPlural(h.k8sResourceClients, "pods")
	err = waitForPodSuccess(podClient, vckName, ns, timeout)
	if err != nil {
		return vckv1.Volume{
			ID:      vc.ID,
			Message: fmt.Sprintf("error during data download using pod [name: %v]: %v", vckName, err),
		}
	}

	return vckv1.Volume{
		ID: vc.ID,
		VolumeSource: corev1.VolumeSource{
			PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
				ClaimName: vckName,
			},
		},
		Message: vckv1.SuccessfulVolumeStatusMessage,
	}
}

func (h *s3DevHandler) OnDelete(ns string, vc vckv1.VolumeConfig, vStatus vckv1.Volume, controllerRef metav1.OwnerReference) {
	podClient := getK8SResourceClientFromPlural(h.k8sResourceClients, "pods")
	podList, err := podClient.List(ns, vc.Labels)
	if err != nil {
		glog.Warningf("[s3-dev-handler] OnDelete: error while listing resource [%s], %v", podClient.Plural(), err)
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
}
