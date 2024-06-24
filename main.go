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

package main

import (
	"context"
	"flag"

	"github.com/IntelAI/vck/pkg/resource/reify"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	vckv1_client "github.com/IntelAI/vck/pkg/client/clientset/versioned"
	"github.com/IntelAI/vck/pkg/controller"
	"github.com/IntelAI/vck/pkg/handlers"
	"github.com/IntelAI/vck/pkg/hooks"
	initializer "github.com/IntelAI/vck/pkg/initializer"
	"github.com/IntelAI/vck/pkg/resource"
	"github.com/IntelAI/vck/pkg/util"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
)

func main() {
	kubeconfig := flag.String("kubeconfig", "", "Path to a kubeconfig file")
	namespace := flag.String("namespace", apiv1.NamespaceAll, "Namespace to monitor (Default all)")
	podTemplateFile := flag.String("podFile", "/etc/volumemanagers/pod.tmpl", "Path to a job template file")
	pachydermPodTemplateFile := flag.String("pachydermPodFile", "/etc/volumemanagers/pod_pachyderm.tmpl", "Path to a job template file for the pachyderm client")
	pvTemplateFile := flag.String("pvFile", "/etc/volumemangers/pv.tmpl", "Path to a job template file")
	pvcTemplateFile := flag.String("pvcFile", "/etc/volumemangers/pvc.tmpl", "Path to a job template file")
	flag.Set("logtostderr", "true")
	flag.Parse()

	config, err := util.BuildConfig(*kubeconfig)
	if err != nil {
		panic(err)
	}

	k8sClientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	crdClient, err := vckv1_client.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	// ApiResources for all the clients we need.
	// NOTE: If new clients are added here, make sure they get reflected in the tests.
	podAPIResource := &metav1.APIResource{
		Kind:       "Pod",
		Name:       "pods",
		Group:      "",
		Version:    "v1",
		Namespaced: true,
	}
	nodeAPIResource := &metav1.APIResource{
		Kind:       "Node",
		Name:       "nodes",
		Group:      "",
		Version:    "v1",
		Namespaced: false,
	}
	pvAPIResource := &metav1.APIResource{
		Kind:       "PersistentVolume",
		Name:       "persistentvolumes",
		Group:      "",
		Version:    "v1",
		Namespaced: false,
	}
	pvcAPIResource := &metav1.APIResource{
		Kind:       "PersistentVolumeClaim",
		Name:       "persistentvolumeclaims",
		Group:      "",
		Version:    "v1",
		Namespaced: true,
	}

	// Since all the clients belong to the same gvk, only one dynamic client is needed in this case.
	config.GroupVersion = &corev1.SchemeGroupVersion
	dynClient, err := dynamic.NewClient(config)
	if err != nil {
		panic(err)
	}

	// Generate runtime.scheme to convert from unstructured to an object.
	corev1Scheme := runtime.NewScheme()
	corev1Scheme.AddKnownTypes(corev1.SchemeGroupVersion, &corev1.PersistentVolume{}, &corev1.Pod{}, &corev1.Node{}, &corev1.PersistentVolumeClaim{})

	reify := &reify.Reify{}
	// The ordering of these resource clients matters. We want the pod to be
	// deployed last as it will use the PVC created before it.
	nodeClient := resource.NewGenericClient(dynClient.Resource(nodeAPIResource, *namespace), "", nodeAPIResource.Name, corev1Scheme, corev1.SchemeGroupVersion, reify)
	pvClient := resource.NewGenericClient(dynClient.Resource(pvAPIResource, *namespace), *pvTemplateFile, pvAPIResource.Name, corev1Scheme, corev1.SchemeGroupVersion, reify)
	pvcClient := resource.NewGenericClient(dynClient.Resource(pvcAPIResource, *namespace), *pvcTemplateFile, pvcAPIResource.Name, corev1Scheme, corev1.SchemeGroupVersion, reify)
	podClient := resource.NewGenericClient(dynClient.Resource(podAPIResource, *namespace), *podTemplateFile, podAPIResource.Name, corev1Scheme, corev1.SchemeGroupVersion, reify)
	pachydermPodClient := resource.NewGenericClient(dynClient.Resource(podAPIResource, *namespace), *pachydermPodTemplateFile, podAPIResource.Name, corev1Scheme, corev1.SchemeGroupVersion, reify)

	dataHandlers := []handlers.DataHandler{
		handlers.NewS3Handler(k8sClientset, []resource.Client{nodeClient, pvClient, pvcClient, podClient, podClient}),
		handlers.NewNFSHandler(k8sClientset, []resource.Client{nodeClient, pvClient, pvcClient, podClient, podClient}),
		handlers.NewPachydermHandler(k8sClientset, []resource.Client{nodeClient, pvClient, pvcClient, pachydermPodClient}),
	}

	// Create hooks
	hooks := hooks.NewVolumeManagerHooks(crdClient.VckV1alpha1().VolumeManagers(*namespace), dataHandlers)

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	// Start a controller for instances of our custom resource.
	controller := controller.New(hooks, crdClient)

	// Start initializer for vck
	initializer := initializer.New(k8sClientset, crdClient)
	go initializer.RunIntializer()

	go controller.Run(ctx, *namespace)

	<-ctx.Done()
}
