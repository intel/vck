package main

import (
	"context"
	"flag"
	"time"

	"github.com/golang/glog"

	apiv1 "k8s.io/api/core/v1"
	extv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	extclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/kubernetes"

	crv1 "github.com/NervanaSystems/kube-controllers-go/cmd/batch-prediction-controller/apis/cr/v1"
	"github.com/NervanaSystems/kube-controllers-go/cmd/batch-prediction-controller/hooks"
	"github.com/NervanaSystems/kube-controllers-go/pkg/controller"
	"github.com/NervanaSystems/kube-controllers-go/pkg/crd"
	"github.com/NervanaSystems/kube-controllers-go/pkg/reconcile"
	"github.com/NervanaSystems/kube-controllers-go/pkg/resource"
	"github.com/NervanaSystems/kube-controllers-go/pkg/util"
)

func main() {
	kubeconfig := flag.String("kubeconfig", "", "Path to a kubeconfig file")
	namespace := flag.String("namespace", apiv1.NamespaceAll, "Namespace to monitor (Default all)")
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

	// k8sclientset, err := kubernetes.NewForConfig(config)
	_, err = kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	// Create new CRD handle for the batch prediction resource type.
	crdHandle := crd.New(
		&crv1.BatchPrediction{},
		&crv1.BatchPredictionList{},
		crv1.GroupName,
		crv1.Version,
		crv1.BatchPredictionResourceKind,
		crv1.BatchPredictionResourceSingular,
		crv1.BatchPredictionResourcePlural,
		extv1beta1.NamespaceScoped,
		*schemaFile,
	)

	err = crd.WriteDefinition(clientset, crdHandle)
	if err != nil {
		// NOTE: We don't panic here, as an existing CRD is absolutely fine.
		// TODO: Validate that the existing CRD is the version we expect.
		glog.Warningf("error while writing %s CRD in namespace %s: %s", crv1.BatchPredictionResourceKind, *namespace, err)
	}

	crdClient, err := crd.NewClient(*config, crdHandle)
	if err != nil {
		panic(err)
	}

	// globalTemplateValues := resource.GlobalTemplateValues{}

	resourceClients := []resource.Client{}

	// Create hooks
	hooks := hooks.NewBatchPredictionHooks(crdClient, resourceClients)

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	// Start a controller for instances of our custom resource.
	controller := controller.New(crdHandle, hooks, crdClient.RESTClient())
	go controller.Run(ctx, *namespace)

	// Start reconciliation in the background for subresource management.
	reconciler := reconcile.New(*namespace, crv1.GVK, crdHandle, crdClient, resourceClients)
	go reconciler.Run(ctx, 10*time.Second)

	<-ctx.Done()
}
