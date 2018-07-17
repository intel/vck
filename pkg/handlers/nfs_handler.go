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
	"fmt"

	"github.com/golang/glog"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/client-go/kubernetes"

	vckv1alpha1 "github.com/IntelAI/vck/pkg/apis/vck/v1alpha1"
	"github.com/IntelAI/vck/pkg/resource"
)

const (
	nfsSourceType vckv1alpha1.DataSourceType = "NFS"
)

type nfsHandler struct {
	sourceType         vckv1alpha1.DataSourceType
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

func (h *nfsHandler) GetSourceType() vckv1alpha1.DataSourceType {
	return h.sourceType
}

func (h *nfsHandler) OnAdd(ns string, vc vckv1.VolumeConfig, controllerRef metav1.OwnerReference) vckv1.Volume {

	if vc.AccessMode != "ReadWriteMany" && vc.AccessMode != "ReadOnlyMany" {
		return vckv1alpha1.Volume{
			ID:      vc.ID,
			Message: fmt.Sprintf("access mode has to be either ReadWriteMany or ReadOnlyMany"),
		}
	}

	vckName := fmt.Sprintf("%s%s", vckNamePrefix, uuid.NewUUID())
	for _, client := range h.k8sResourceClients {
		if client.Plural() == "nodes" || client.Plural() == "pods" {
			continue
		}

		err := client.Create(ns, struct {
			vckv1alpha1.VolumeConfig
			metav1.OwnerReference
			NS                  string
			NodeName            string
			VCKName             string
			VCKStorageClassName string
			PVType              string
			VCKOptions          map[string]string
		}{
			vc,
			controllerRef,
			ns,
			"",
			vckName,
			"vck",
			"nfs",
			map[string]string{
				"server": vc.Options["server"],
				"path":   vc.Options["path"],
			},
		})

		if err != nil {
			return vckv1alpha1.Volume{
				ID:      vc.ID,
				Message: fmt.Sprintf("error during sub-resource [%s] creation: %v", client.Plural(), err),
			}
		}
	}

	return vckv1alpha1.Volume{
		ID: vc.ID,
		VolumeSource: corev1.VolumeSource{
			PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
				ClaimName: vckName,
			},
		},
		Message: vckv1alpha1.SuccessfulVolumeStatusMessage,
	}
}

func (h *nfsHandler) OnDelete(ns string, vc vckv1alpha1.VolumeConfig, vStatus vckv1alpha1.Volume, controllerRef metav1.OwnerReference) {
	for _, client := range h.k8sResourceClients {
		if client.Plural() == "nodes" || client.Plural() == "pods" {
			continue
		}

		resourceList, err := client.List(ns, vc.Labels)
		if err != nil {
			glog.Warningf("[nfs-handler] OnDelete: error while listing resource [%s], %v", client.Plural(), err)
		}

		for _, resource := range resourceList {
			resControllerRef := metav1.GetControllerOf(resource)
			if resControllerRef == nil {
				continue
			}

			if resControllerRef.UID == controllerRef.UID {
				client.Delete(ns, resource.GetName())
			}
		}
	}

}
