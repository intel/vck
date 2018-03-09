package main

import (
	"context"
	"flag"
	"github.com/NervanaSystems/kube-volume-controller/pkg/resource/reify"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	kvcv1_client "github.com/NervanaSystems/kube-volume-controller/pkg/client/clientset/versioned"
	"github.com/NervanaSystems/kube-volume-controller/pkg/controller"
	"github.com/NervanaSystems/kube-volume-controller/pkg/handlers"
	"github.com/NervanaSystems/kube-volume-controller/pkg/hooks"
	"github.com/NervanaSystems/kube-volume-controller/pkg/resource"
	"github.com/NervanaSystems/kube-volume-controller/pkg/util"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
)

func main() {
	kubeconfig := flag.String("kubeconfig", "", "Path to a kubeconfig file")
	namespace := flag.String("namespace", apiv1.NamespaceAll, "Namespace to monitor (Default all)")
	podTemplateFile := flag.String("podFile", "/etc/volumemanagers/pod.tmpl", "Path to a job template file")
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

	crdClient, err := kvcv1_client.NewForConfig(config)
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
	resourceClients := []resource.Client{
		resource.NewGenericClient(dynClient.Resource(nodeAPIResource, *namespace), "", nodeAPIResource.Name, corev1Scheme, corev1.SchemeGroupVersion, reify),
		resource.NewGenericClient(dynClient.Resource(pvAPIResource, *namespace), *pvTemplateFile, pvAPIResource.Name, corev1Scheme, corev1.SchemeGroupVersion, reify),
		resource.NewGenericClient(dynClient.Resource(pvcAPIResource, *namespace), *pvcTemplateFile, pvcAPIResource.Name, corev1Scheme, corev1.SchemeGroupVersion, reify),
		resource.NewGenericClient(dynClient.Resource(podAPIResource, *namespace), *podTemplateFile, podAPIResource.Name, corev1Scheme, corev1.SchemeGroupVersion, reify),
	}

	dataHandlers := []handlers.DataHandler{
		handlers.NewS3Handler(k8sClientset, resourceClients),
		handlers.NewS3DevHandler(k8sClientset, resourceClients),
		handlers.NewNFSHandler(k8sClientset, resourceClients),
	}

	// Create hooks
	hooks := hooks.NewVolumeManagerHooks(crdClient.KvcV1().VolumeManagers(*namespace), dataHandlers)

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	// Start a controller for instances of our custom resource.
	controller := controller.New(hooks, crdClient)
	go controller.Run(ctx, *namespace)

	<-ctx.Done()
}
