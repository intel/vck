/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Note: the example only works with the code within the same release/branch.
package main

import (
	"context"
	"flag"
	"fmt"
	"time"

	apiv1 "k8s.io/api/core/v1"
	extv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	extclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/rest"
	// Uncomment the following line to load the gcp plugin (only required to
	// authenticate against GKE clusters).
	// _ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	crv1 "github.com/NervanaSystems/kube-controllers-go/cmd/example-controller/apis/cr/v1"
	"github.com/NervanaSystems/kube-controllers-go/pkg/controller"
	"github.com/NervanaSystems/kube-controllers-go/pkg/crd"
	"github.com/NervanaSystems/kube-controllers-go/pkg/states"
	"github.com/NervanaSystems/kube-controllers-go/pkg/util"
)

func main() {
	kubeconfig := flag.String("kubeconfig", "", "Path to a kube config. Only required if out-of-cluster.")
	flag.Parse()

	// Create the client config. Use kubeconfig if given, otherwise assume
	// in-cluster.
	config, err := util.BuildConfig(*kubeconfig)
	if err != nil {
		panic(err)
	}

	clientset, err := extclient.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	// Create new CRD handle for the example resource type.
	crdHandle := crd.New(
		&crv1.Example{},
		&crv1.ExampleList{},
		crv1.GroupName,
		crv1.Version,
		crv1.ExampleResourceKind,
		crv1.ExampleResourceSingular,
		crv1.ExampleResourcePlural,
		extv1beta1.NamespaceScoped,
		"",
	)

	// Initialize custom resource using a CustomResourceDefinition if it does
	// not exist
	err = crd.WriteDefinition(clientset, crdHandle)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		panic(err)
	}

	// NB: This is ONLY for the example controller. A CR's definition ought not
	// be deleted when a controller stops in a production environment.
	defer crd.DeleteDefinition(clientset, crdHandle)

	// Make a new config for our extension's API group, using the first config
	// as a baseline
	crdClient, err := crd.NewClient(*config, crdHandle)
	if err != nil {
		panic(err)
	}

	// Start a controller for instances of our custom resource.
	controller := controller.New(crdHandle, &exampleHooks{crdClient}, crdClient.RESTClient())

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	go controller.Run(ctx, apiv1.NamespaceAll)

	// Create an instance of our custom resource.
	example := &crv1.Example{
		ObjectMeta: metav1.ObjectMeta{
			Name: "example1",
		},
		Spec: crv1.ExampleSpec{
			Foo: "hello",
			Bar: true,
		},
		Status: crv1.ExampleStatus{
			State:   states.Pending,
			Message: "Created, not processed yet",
		},
	}

	// TODO(CD): Replace this rest client usage with the CRD client wrapper
	//           when the Create method supports returning the result object.
	var result crv1.Example
	err = crdClient.RESTClient().Post().
		Resource(crv1.ExampleResourcePlural).
		Namespace(apiv1.NamespaceDefault).
		Body(example).
		Do().Into(&result)
	if err == nil {
		fmt.Printf("CREATED: %#v\n", result)
	} else if apierrors.IsAlreadyExists(err) {
		fmt.Printf("ALREADY EXISTS: %#v\n", result)
	} else {
		panic(err)
	}

	// Poll until Example object is handled by controller and gets status updated
	// to "Processed"
	err = waitForExampleInstanceProcessed(crdClient.RESTClient(), "example1")
	if err != nil {
		panic(err)
	}
	fmt.Print("PROCESSED\n")

	// Fetch a list of our CRs
	exampleList := crv1.ExampleList{}
	err = crdClient.RESTClient().Get().Resource(crv1.ExampleResourcePlural).Do().Into(&exampleList)
	if err != nil {
		panic(err)
	}
	fmt.Printf("LIST: %#v\n", exampleList)

	<-ctx.Done()
}

func waitForExampleInstanceProcessed(crdClient rest.Interface, name string) error {
	return wait.Poll(100*time.Millisecond, 10*time.Second, func() (bool, error) {
		var example crv1.Example
		err := crdClient.Get().
			Resource(crv1.ExampleResourcePlural).
			Namespace(apiv1.NamespaceDefault).
			Name(name).
			Do().Into(&example)

		if err == nil && example.Status.State == states.Completed {
			return true, nil
		}

		return false, err
	})
}
