//
// Copyright (c) 2018 Intel Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// SPDX-License-Identifier: EPL-2.0
//

package handlers

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/client-go/kubernetes"

	"github.com/golang/glog"

	"bytes"
	vckv1alpha1 "github.com/IntelAI/vck/pkg/apis/vck/v1alpha1"
	"github.com/IntelAI/vck/pkg/resource"
)

const (
	s3SourceType vckv1alpha1.DataSourceType = "S3"
)

type s3Handler struct {
	sourceType         vckv1alpha1.DataSourceType
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

func (h *s3Handler) GetSourceType() vckv1alpha1.DataSourceType {
	return h.sourceType
}

func (h *s3Handler) OnAdd(ns string, vc vckv1alpha1.VolumeConfig, controllerRef metav1.OwnerReference) vckv1alpha1.Volume {
	if len(vc.Labels) == 0 {
		return vckv1alpha1.Volume{
			ID:      vc.ID,
			Message: fmt.Sprintf("labels cannot be empty"),
		}
	}

	if _, ok := vc.Options["awsCredentialsSecretName"]; !ok {
		return vckv1alpha1.Volume{
			ID:      vc.ID,
			Message: fmt.Sprintf("awsCredentialsSecretName key has to be set in options"),
		}
	}

	if vc.AccessMode != "ReadWriteOnce" {
		return vckv1alpha1.Volume{
			ID:      vc.ID,
			Message: fmt.Sprintf("access mode has to be ReadWriteOnce"),
		}
	}

	if _, ok := vc.Options["sourceURL"]; !ok {
		return vckv1alpha1.Volume{
			ID:      vc.ID,
			Message: fmt.Sprintf("sourceURL has to be set in options"),
		}
	}

	if _, ok := vc.Options["endpointURL"]; !ok {
		vc.Options["EndpointURL"] = "https://s3.amazonaws.com"
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
			return vckv1alpha1.Volume{
				ID:      vc.ID,
				Message: fmt.Sprintf("error while parsing timeout for data download: %v", err),
			}
		}
	}

	nodeClient := getK8SResourceClientFromPlural(h.k8sResourceClients, "nodes")
	nodeList, err := nodeClient.List(ns, map[string]string{})
	if err != nil {
		return vckv1alpha1.Volume{
			ID:      vc.ID,
			Message: fmt.Sprintf("error getting node list: %v", err),
		}
	}

	// If number of nodes < replicas, then return immediately.
	if len(nodeList) < vc.Replicas {
		return vckv1alpha1.Volume{
			ID: vc.ID,
			Message: fmt.Sprintf("replicas [%v] greater than number of nodes [%v]",
				vc.Replicas, len(nodeList)),
		}
	}

	vckDataPathSuffix := fmt.Sprintf("%s%s", vckNamePrefix, uuid.NewUUID())
	vckPath := fmt.Sprintf("%s/%s", vc.Options["dataPath"], vckDataPathSuffix)
	copyCommand := []string{}

	if distributionStrategy, ok := vc.Options["distributionStrategy"]; ok {
		var distributionMap map[string]int

		err := json.Unmarshal([]byte(distributionStrategy), &distributionMap)
		if err != nil {
			return vckv1alpha1.Volume{
				ID:      vc.ID,
				Message: fmt.Sprintf("invalid distributionStrategy [%v] specified, it must be a map[string]int", distributionStrategy),
			}
		}
		replicaCount := 0

		for filter, replicas := range distributionMap {

			for i := 0; i < replicas; i++ {
				copyCommand = append(copyCommand, fmt.Sprintf("mc config host add s3 ${AWS_ENDPOINT_URL} ${AWS_ACCESS_KEY_ID} ${AWS_SECRET_ACCESS_KEY}; mc find s3/${BUCKET_NAME}${BUCKET_PATH} --path '%v' --exec 'mc cp {} %s'", filter, vckPath))
				replicaCount++
			}
		}
		if replicaCount != vc.Replicas {
			return vckv1alpha1.Volume{
				ID:      vc.ID,
				Message: fmt.Sprintf("total number of replicas: [%v] in distributionStrategy [%v], does not match number or replicas provided: [%v]", replicaCount, distributionStrategy, vc.Replicas),
			}
		}
	} else {
		for i := 0; i < vc.Replicas; i++ {
			copyCommand = append(copyCommand, "mc config host add s3 ${AWS_ENDPOINT_URL} ${AWS_ACCESS_KEY_ID} ${AWS_SECRET_ACCESS_KEY}; mc cp ${RECURSIVE_OPTION} s3/${BUCKET_NAME}${BUCKET_PATH} ${DATA_PATH}")
		}
	}

	recursiveFlag := ""
	if strings.HasSuffix(vc.Options["sourceURL"], "/") {
		recursiveFlag = "--recursive"
	}

	s3URL, err := url.Parse(vc.Options["sourceURL"])
	if err != nil {
		return vckv1alpha1.Volume{
			ID:      vc.ID,
			Message: fmt.Sprintf("error while parsing URL [%s]: %v", vc.Options["sourceURL"], err),
		}
	}
	bucketName := s3URL.Host
	bucketPath := s3URL.Path

	vckNames := []string{}
	podClient := getK8SResourceClientFromPlural(h.k8sResourceClients, "pods")
	for i := 0; i < vc.Replicas; i++ {
		vckName := fmt.Sprintf("%s%s", vckNamePrefix, uuid.NewUUID())
		vckNames = append(vckNames, vckName)

		err = podClient.Create(ns, struct {
			vckv1alpha1.VolumeConfig
			metav1.OwnerReference
			NS              string
			VCKName         string
			VCKOp           string
			RecursiveOption string
			BucketName      string
			BucketPath      string
			VCKOptions      map[string]string
		}{
			vc,
			controllerRef,
			ns,
			vckName,
			"add",
			recursiveFlag,
			bucketName,
			bucketPath,
			map[string]string{
				"path":        vckPath,
				"copyCommand": copyCommand[i],
			},
		})

		if err != nil {
			return vckv1alpha1.Volume{
				ID:      vc.ID,
				Message: fmt.Sprintf("error during sub-resource [%s] creation: %v", podClient.Plural(), err),
			}
		}
	}

	usedNodeNames := []string{}
	nodeLabelKey := fmt.Sprintf("%s/%s-%s-%s", vckv1alpha1.GroupName, ns, controllerRef.Name, vc.ID)
	for _, vckName := range vckNames {
		err := waitForPodSuccess(podClient, vckName, ns, timeout)

		if err != nil {
			downloadErrMsg := "error during data download using pod"

			podResource := h.k8sClientset.CoreV1().RESTClient().Get().Namespace(ns).Name(vckName).Resource("pods")
			if podResource == nil {
				return vckv1alpha1.Volume{
					ID:      vc.ID,
					Message: fmt.Sprintf("%v [name: %v]: %v", downloadErrMsg, vckName, err),
				}
			}

			logReq := podResource.SubResource("log")
			if logReq == nil {
				return vckv1alpha1.Volume{
					ID:      vc.ID,
					Message: fmt.Sprintf("%v [name: %v]: %v", downloadErrMsg, vckName, err),
				}
			}

			readCloser, logErr := logReq.Stream()
			if logErr != nil {
				return vckv1alpha1.Volume{
					ID:      vc.ID,
					Message: fmt.Sprintf("%v [name: %v]: %v", downloadErrMsg, vckName, err),
				}
			}

			defer readCloser.Close()
			logBuf := new(bytes.Buffer)
			logBuf.ReadFrom(readCloser)
			return vckv1alpha1.Volume{
				ID:      vc.ID,
				Message: fmt.Sprintf("%v [name: %v]: %v", downloadErrMsg, vckName, logBuf.String()),
			}
		}

		podObj, err := podClient.Get(ns, vckName)
		if err != nil {
			return vckv1alpha1.Volume{
				ID:      vc.ID,
				Message: fmt.Sprintf("error getting pod [name: %v]: %v", vckName, err),
			}
		}

		pod, ok := podObj.(*corev1.Pod)
		if !ok {
			return vckv1alpha1.Volume{
				ID:      vc.ID,
				Message: fmt.Sprintf("object returned from podclient.Get() is not a pod"),
			}
		}

		usedNodeNames = append(usedNodeNames, pod.Spec.NodeName)

		node, err := nodeClient.Get("", pod.Spec.NodeName)
		if err != nil {
			return vckv1alpha1.Volume{
				ID:      vc.ID,
				Message: fmt.Sprintf("could not get node %s, error: %v", pod.Spec.NodeName, err),
			}
		}
		// update nodes with the correct label
		err = updateNodeWithLabels(nodeClient, node.(*corev1.Node), []string{nodeLabelKey}, "add")

		if err != nil {
			return vckv1alpha1.Volume{
				ID:      vc.ID,
				Message: fmt.Sprintf("could not label node %s, error: %v", pod.Spec.NodeName, err),
			}
		}

	}

	return vckv1alpha1.Volume{
		ID: vc.ID,
		VolumeSource: corev1.VolumeSource{
			HostPath: &corev1.HostPathVolumeSource{
				Path: fmt.Sprintf("%s/%s", vc.Options["dataPath"], vckDataPathSuffix),
			},
		},
		NodeAffinity: corev1.NodeAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
				NodeSelectorTerms: []corev1.NodeSelectorTerm{
					{
						MatchExpressions: []corev1.NodeSelectorRequirement{
							{
								Key:      nodeLabelKey,
								Operator: corev1.NodeSelectorOpExists,
							},
						},
					},
				},
			},
		},
		Message: vckv1alpha1.SuccessfulVolumeStatusMessage,
	}
}

func (h *s3Handler) OnDelete(ns string, vc vckv1alpha1.VolumeConfig, vStatus vckv1alpha1.Volume, controllerRef metav1.OwnerReference) {
	nodeLabelKey := fmt.Sprintf("%s/%s-%s-%s", vckv1alpha1.GroupName, ns, controllerRef.Name, vc.ID)
	podClient := getK8SResourceClientFromPlural(h.k8sResourceClients, "pods")

	if vStatus.VolumeSource != (corev1.VolumeSource{}) {
		vckNames := []string{}
		for i := 0; i < vc.Replicas; i++ {
			vckName := fmt.Sprintf("%s%s", vckNamePrefix, uuid.NewUUID())
			vckNames = append(vckNames, vckName)

			err := podClient.Create(ns, struct {
				vckv1alpha1.VolumeConfig
				metav1.OwnerReference
				NS              string
				VCKName         string
				VCKOp           string
				VCKNodeLabelKey string
				VCKOptions      map[string]string
			}{
				vc,
				controllerRef,
				ns,
				vckName,
				"delete",
				nodeLabelKey,
				map[string]string{
					"path": vStatus.VolumeSource.HostPath.Path,
				},
			})

			if err != nil {
				glog.Warningf("error during sub-resource [%s] deletion: %v", podClient.Plural(), err)
			}
		}

		timeout, _ := time.ParseDuration("3m")
		for _, vckName := range vckNames {
			err := waitForPodSuccess(podClient, vckName, ns, timeout)
			if err != nil {
				// TODO(balajismaniam): append pod logs to this message if possible.
				glog.Warningf("error during data deletion using pod [name: %v]: %v", vckName, err)
			}
			podClient.Delete(ns, vckName)
		}
	}

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

	// Delete the label for the node
	nodeClient := getK8SResourceClientFromPlural(h.k8sResourceClients, "nodes")

	// Get the node list based on the label
	nodeList, err := nodeClient.List("", map[string]string{nodeLabelKey: "true"})
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

		err = updateNodeWithLabels(nodeClient, node.(*corev1.Node), []string{nodeLabelKey}, "delete")
		if err != nil {
			glog.Warningf("[s3-handler] OnDelete: error while deleting label for node nodes %v", err)
		}
	}
}
