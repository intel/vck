package initialzer

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	vckv1 "github.com/IntelAI/vck/pkg/apis/vck/v1"
	vckv1_client "github.com/IntelAI/vck/pkg/client/clientset/versioned"
	state "github.com/IntelAI/vck/pkg/states"
	"k8s.io/api/apps/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

const (
	annotation      = "initializer.kubernetes.io/vck"
	initializerName = "vck.initializer.kubernetes.io"
)

type data struct {
	Name       string       `json:"name"`
	ID         string       `json:"id,omitempty"`
	Containers []containers `json:"containers,omitempty"`
}

type containers struct {
	Name      string `json:"name"`
	MountPath string `json:"mount-path,omitempty"`
}

type config struct {
	Containers []corev1.Container
	Volumes    []corev1.Volume
}

// Initializer watches a deployment and delegates create events
// to a set of supplied callback functions.
type Initializer struct {
	ClientSet *kubernetes.Clientset
	CRDClient *vckv1_client.Clientset
}

// New returns a new Initializer.
func New(clientset *kubernetes.Clientset, crdClient *vckv1_client.Clientset) *Initializer {
	return &Initializer{
		ClientSet: clientset,
		CRDClient: crdClient,
	}
}

// RunIntializer starts a vck inistializer
func (i *Initializer) RunIntializer() {
	restClient := i.ClientSet.AppsV1beta1().RESTClient()
	watchlist := cache.NewListWatchFromClient(restClient, "deployments", corev1.NamespaceAll, fields.Everything())
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
				err := initializeDeployment(o, i)
				if err != nil {
					log.Println(err)
					log.Println("Deleteting Deployment " + o.Name)
					deletePolicy := metav1.DeletePropagationBackground
					err := i.ClientSet.AppsV1().Deployments(o.Namespace).Delete(o.Name, &metav1.DeleteOptions{
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
func initializeDeployment(deployment *v1beta1.Deployment, initializer *Initializer) error {
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
			a := deployment.ObjectMeta.GetAnnotations()
			_, ok := a[annotation]
			if !ok {
				log.Printf("Required '%s' annotation missing; skipping vck initializing", annotation)
				_, err := initializer.ClientSet.AppsV1beta1().Deployments(deployment.Namespace).Update(initializedDeployment)
				if err != nil {
					return err
				}
				return nil
			}
			//log.Print("annotation: ", a[annotation])
			infoArray := make([]data, 0)
			nodeSelectorTermArr := make([]corev1.NodeSelectorTerm, 0)
			err := json.Unmarshal([]byte(a[annotation]), &infoArray)
			if err != nil {
				return err
			}
			fmt.Println("Unmarshal:", infoArray[0].Name)
			for _, info := range infoArray {

				vckVM, err := initializer.CRDClient.VckV1().VolumeManagers(deployment.GetNamespace()).Get(info.Name, metav1.GetOptions{})
				if err != nil {
					return err
				}
				if vckVM.Status.State != state.Running && len(vckVM.Status.Volumes) == 0 {
					return errors.New("given vck is not in usable state " + string(vckVM.Status.State))
				}
				volumeVCK, nodeSelectorTerm, err := getVolumesAffinity(vckVM, &info)
				if err != nil {
					return err
				}
				if len(info.Containers) == 0 {
					for _, container := range deployment.Spec.Template.Spec.Containers {
						tempContainer := containers{
							Name:      container.Name,
							MountPath: "/var/datasets",
						}
						info.Containers = append(info.Containers, tempContainer)
					}
				}
				for _, container := range info.Containers {
					if info.ID == "" {
						info.ID = vckVM.Status.Volumes[0].ID
					}
					volumeMount, containerID, err := addVolumeMount(deployment, info.Name+info.ID, container)
					if err != nil {
						return err
					}

					initializedDeployment.Spec.Template.Spec.Containers[containerID].VolumeMounts = append(initializedDeployment.Spec.Template.Spec.Containers[containerID].VolumeMounts, *volumeMount)
				}
				found := false
				for _, item := range initializedDeployment.Spec.Template.Spec.Volumes {
					if item.Name == volumeVCK.Name {
						found = true
						break
					}
				}
				if !found {
					initializedDeployment.Spec.Template.Spec.Volumes = append(initializedDeployment.Spec.Template.Spec.Volumes, *volumeVCK)
				}
				found = false
				for _, item1 := range nodeSelectorTermArr {
					for _, item2 := range *nodeSelectorTerm {
						if item1.MatchExpressions[0].Key == item2.MatchExpressions[0].Key {
							found = true
							break
						}
					}
					if found {
						break
					}
				}
				if !found {
					nodeSelectorTermArr = append(nodeSelectorTermArr, *nodeSelectorTerm...)
				}

			}

			vckAffinity := corev1.Affinity{
				NodeAffinity: &corev1.NodeAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
						NodeSelectorTerms: nodeSelectorTermArr,
					},
				},
			}

			initializedDeployment.Spec.Template.Spec.Affinity = &vckAffinity

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

			_, err = initializer.ClientSet.AppsV1beta1().Deployments(deployment.Namespace).Patch(deployment.Name, types.StrategicMergePatchType, patchBytes)
			if err != nil {
				return err
			}

		}
	}
	return nil
}

func getVolumesAffinity(vckVM *vckv1.VolumeManager, info *data) (*corev1.Volume, *[]corev1.NodeSelectorTerm, error) {
	if len(info.ID) == 0 {
		item := vckVM.Status.Volumes[0]
		volumeVCK := corev1.Volume{
			Name: info.Name + vckVM.Status.Volumes[0].ID,
			VolumeSource: corev1.VolumeSource{
				HostPath: item.VolumeSource.HostPath,
			},
		}
		nodeSelectorTerm := vckVM.Status.Volumes[0].NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms
		log.Print("matchExpression: ", nodeSelectorTerm)
		return &volumeVCK, &nodeSelectorTerm, nil
	}
	for i, item := range vckVM.Status.Volumes {
		if info.ID == item.ID {
			volumeVCK := corev1.Volume{
				Name: info.Name + info.ID,
				VolumeSource: corev1.VolumeSource{
					HostPath: item.VolumeSource.HostPath,
				},
			}
			nodeSelectorTerm := vckVM.Status.Volumes[i].NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms
			log.Print("matchExpression: ", nodeSelectorTerm)
			return &volumeVCK, &nodeSelectorTerm, nil
		}
	}
	return nil, nil, errors.New("given id for vck does not exists")

}

func addVolumeMount(deployment *v1beta1.Deployment, name string, container containers) (*corev1.VolumeMount, int, error) {
	containerID := -1
	if len(container.MountPath) == 0 {
		container.MountPath = "/var/datasets"
	}
	for id, item := range deployment.Spec.Template.Spec.Containers {
		if container.Name == item.Name {
			containerID = id
			volumeMount := corev1.VolumeMount{
				MountPath: container.MountPath,
				Name:      name,
			}
			return &volumeMount, containerID, nil
		}
	}
	return nil, -1, errors.New("given container name for vck does not exists ")
}
