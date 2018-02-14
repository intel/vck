package main

import (
	"context"
	"flag"

	"github.com/golang/glog"

	apiv1 "k8s.io/api/core/v1"
	extv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	extclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/kubernetes"

	"github.com/NervanaSystems/kube-controllers-go/pkg/controller"
	"github.com/NervanaSystems/kube-controllers-go/pkg/crd"
	"github.com/NervanaSystems/kube-controllers-go/pkg/resource"
	"github.com/NervanaSystems/kube-controllers-go/pkg/util"
	crv1 "github.com/NervanaSystems/kube-volume-controller/pkg/apis/cr/v1"
	"github.com/NervanaSystems/kube-volume-controller/pkg/handlers"
	"github.com/NervanaSystems/kube-volume-controller/pkg/hooks"
)

func main() {
	kubeconfig := flag.String("kubeconfig", "", "Path to a kubeconfig file")
	namespace := flag.String("namespace", apiv1.NamespaceAll, "Namespace to monitor (Default all)")
	podTemplateFile := flag.String("podFile", "/etc/volumemanagers/pod.tmpl", "Path to a job template file")
	pvTemplateFile := flag.String("pvFile", "/etc/volumemangers/pv.tmpl", "Path to a job template file")
	pvcTemplateFile := flag.String("pvcFile", "/etc/volumemangers/pvc.tmpl", "Path to a job template file")
	schemaFile := flag.String("schema", "", "Path to a custom resource schema file")
	flag.Set("logtostderr", "true")
	flag.Parse()

	config, err := util.BuildConfig(*kubeconfig)
	if err != nil {
		panic(err)
	}

	clientset, err := extclient.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	k8sClientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	// Create new CRD handle for the volume manager resource type.
	crdHandle := crd.New(
		&crv1.VolumeManager{},
		&crv1.VolumeManagerList{},
		crv1.GroupName,
		crv1.Version,
		crv1.VolumeManagerResourceKind,
		crv1.VolumeManagerResourceSingular,
		crv1.VolumeManagerResourcePlural,
		extv1beta1.NamespaceScoped,
		*schemaFile,
	)

	err = crd.WriteDefinition(clientset, crdHandle)
	if err != nil {
		// NOTE: We don't panic here, as an existing CRD is absolutely fine.
		// TODO: Validate that the existing CRD is the version we expect.
		glog.Warningf("error while writing %s CRD in namespace %s: %s", crv1.VolumeManagerResourceKind, *namespace, err)
	}

	crdClient, err := crd.NewClient(*config, crdHandle)
	if err != nil {
		panic(err)
	}

	globalTemplateValues := resource.GlobalTemplateValues{}
	// The ordering of these resource clients matters. We want the pod to be
	// deployed last as it will use the PVC created before it.
	resourceClients := []resource.Client{
		resource.NewNodeClient(globalTemplateValues, k8sClientset, ""),
		resource.NewPersistentVolumeClient(globalTemplateValues, k8sClientset, *pvTemplateFile),
		resource.NewPersistentVolumeClaimClient(globalTemplateValues, k8sClientset, *pvcTemplateFile),
		resource.NewPodClient(globalTemplateValues, k8sClientset, *podTemplateFile),
	}

	dataHandlers := []handlers.DataHandler{
		handlers.NewS3Handler(k8sClientset, resourceClients),
		handlers.NewS3DevHandler(k8sClientset, resourceClients),
		handlers.NewNFSHandler(k8sClientset, resourceClients),
	}

	// Create hooks
	hooks := hooks.NewVolumeManagerHooks(crdClient, dataHandlers)

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	// Start a controller for instances of our custom resource.
	controller := controller.New(crdHandle, hooks, crdClient.RESTClient())
	go controller.Run(ctx, *namespace)

	<-ctx.Done()
}
