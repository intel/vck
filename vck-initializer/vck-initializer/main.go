// Copyright 2017 Google Inc. All Rights Reserved.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//     http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	vckv1 "github.com/IntelAI/vck/pkg/apis/vck/v1"
	vckv1_client "github.com/IntelAI/vck/pkg/client/clientset/versioned"
	"k8s.io/api/apps/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

const (
	defaultAnnotation      = "initializer.kubernetes.io/vck"
	defaultInitializerName = "vck.initializer.kubernetes.io"
	defaultNamespace       = "vck"
)

var (
	annotation        string
	initializerName   string
	namespace         string
	requireAnnotation bool
)

type data struct {
	Name       string   `json:"name"`
	ID         string   `json:"id"`
	Containers []string `json:"containers,omitempty"`
	MountPath  string   `json:"mount-path"`
}

type config struct {
	Containers []corev1.Container
	Volumes    []corev1.Volume
}

func main() {
	flag.StringVar(&annotation, "annotation", defaultAnnotation, "The annotation to trigger initialization")
	flag.StringVar(&initializerName, "initializer-name", defaultInitializerName, "The initializer name")
	flag.StringVar(&namespace, "namespace", defaultNamespace, "The configuration namespace")
	flag.BoolVar(&requireAnnotation, "require-annotation", true, "Require annotation for initialization")
	flag.Parse()

	log.Println("Starting the Kubernetes initializer...")
	log.Printf("Initializer name set to: %s", initializerName)

	clusterConfig, err := rest.InClusterConfig()
	if err != nil {
		log.Fatal(err.Error())
	}

	clientset, err := kubernetes.NewForConfig(clusterConfig)
	if err != nil {
		log.Fatal(err)
	}
	crdClient, err := vckv1_client.NewForConfig(clusterConfig)
	if err != nil {
		panic(err)
	}

	// Watch uninitialized Deployments in all namespaces.
	restClient := clientset.AppsV1beta1().RESTClient()
	watchlist := cache.NewListWatchFromClient(restClient, "deployments", corev1.NamespaceAll, fields.Everything())

	// Wrap the returned watchlist to workaround the inability to include
	// the `IncludeUninitialized` list option when setting up watch clients.
	includeUninitializedWatchlist := &cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
			options.IncludeUninitialized = true
			return watchlist.List(options)
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			options.IncludeUninitialized = true
			return watchlist.Watch(options)
		},
	}

	resyncPeriod := 30 * time.Second

	_, controller := cache.NewInformer(includeUninitializedWatchlist, &v1beta1.Deployment{}, resyncPeriod,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				o := obj.(*v1beta1.Deployment)
				err := initializeDeployment(o, clientset, crdClient)
				if err != nil {
					log.Println(err)
					log.Println("Deleteting Deployment " + o.Name)
					deletePolicy := metav1.DeletePropagationBackground
					err := clientset.AppsV1().Deployments(o.Namespace).Delete(o.Name, &metav1.DeleteOptions{
						PropagationPolicy: &deletePolicy,
					})
					if err != nil {
						panic(err)
					}
					log.Println("Deleted Deployment.")

				}
			},
		},
	)

	stop := make(chan struct{})
	go controller.Run(stop)

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	<-signalChan

	log.Println("Shutdown signal received, exiting...")
	close(stop)
}

func initializeDeployment(deployment *v1beta1.Deployment, clientset *kubernetes.Clientset, crdClient *vckv1_client.Clientset) error {
	if deployment.ObjectMeta.GetInitializers() != nil {
		pendingInitializers := deployment.ObjectMeta.GetInitializers().Pending

		if initializerName == pendingInitializers[0].Name {
			log.Printf("Initializing deployment: %s", deployment.Name)

			o := deployment.DeepCopyObject()
			initializedDeployment := o.(*v1beta1.Deployment)

			// Remove self from the list of pending Initializers while preserving ordering.
			if len(pendingInitializers) == 1 {
				initializedDeployment.ObjectMeta.Initializers = nil
			} else {
				initializedDeployment.ObjectMeta.Initializers.Pending = append(pendingInitializers[:0], pendingInitializers[1:]...)
			}
			if requireAnnotation {
				a := deployment.ObjectMeta.GetAnnotations()
				_, ok := a[annotation]
				if !ok {
					log.Printf("Required '%s' annotation missing; skipping vck initializing", annotation)
					_, err := clientset.AppsV1beta1().Deployments(deployment.Namespace).Update(initializedDeployment)
					if err != nil {
						return err
					}
					return nil
				}
				//log.Print("annotation: ", a[annotation])
				info := &data{}
				err := json.Unmarshal([]byte(a[annotation]), info)
				if err != nil {
					return err
				}
				fmt.Println("Unmarshal:", info.MountPath)
				vckVM, err := crdClient.VckV1().VolumeManagers(deployment.GetNamespace()).Get(info.Name, metav1.GetOptions{})
				if err != nil {
					return err
				}
				volumeVCK, affinityVCK, err := addVolumesAffinity(vckVM, info)
				if err != nil {
					return err
				}
				if info.Containers == nil {
					for _, container := range deployment.Spec.Template.Spec.Containers {
						info.Containers = append(info.Containers, container.Name)
					}
				}

				for _, container := range info.Containers {
					volumeMount, containerID, err := addVolumeMount(deployment, vckVM, container, info.MountPath)
					if err != nil {
						return err
					}
					initializedDeployment.Spec.Template.Spec.Containers[containerID].VolumeMounts = append(deployment.Spec.Template.Spec.Containers[containerID].VolumeMounts, *volumeMount)
				}
				initializedDeployment.Spec.Template.Spec.Volumes = append(deployment.Spec.Template.Spec.Volumes, *volumeVCK)
				initializedDeployment.Spec.Template.Spec.Affinity = affinityVCK

			}
			oldData, err := json.Marshal(deployment)
			if err != nil {
				return err
			}

			newData, err := json.Marshal(initializedDeployment)
			if err != nil {
				return err
			}

			patchBytes, err := strategicpatch.CreateTwoWayMergePatch(oldData, newData, v1beta1.Deployment{})
			if err != nil {
				return err
			}

			_, err = clientset.AppsV1beta1().Deployments(deployment.Namespace).Patch(deployment.Name, types.StrategicMergePatchType, patchBytes)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func addVolumesAffinity(vckVM *vckv1.VolumeManager, info *data) (*corev1.Volume, *corev1.Affinity, error) {
	for _, item := range vckVM.Status.Volumes {
		if info.ID == item.ID {
			volumeVCK := corev1.Volume{
				Name: "dataset-claim",
				VolumeSource: corev1.VolumeSource{
					HostPath: item.VolumeSource.HostPath,
				},
			}
			affinityVCK := corev1.Affinity{
				NodeAffinity: &item.NodeAffinity,
			}
			return &volumeVCK, &affinityVCK, nil
		}
	}
	return nil, nil, errors.New("given id for vck does not exists")

}

func addVolumeMount(deployment *v1beta1.Deployment, vckVM *vckv1.VolumeManager, container string, mountPath string) (*corev1.VolumeMount, int, error) {
	containerID := -1
	for id, item := range deployment.Spec.Template.Spec.Containers {
		if container == item.Name {
			containerID = id
			volumeMount := corev1.VolumeMount{
				MountPath: mountPath,
				Name:      "dataset-claim",
			}
			return &volumeMount, containerID, nil
		}
	}
	return nil, -1, errors.New("given container name for vck does not exists ")
}
