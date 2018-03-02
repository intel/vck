# Developer Manual: Kubernetes Volume Controller (KVC)
  
  * [Developer Manual: Kubernetes Volume Controller (KVC)](#developer-manual-kubernetes-volume-controller-kvc)
    * [Testing and Building](#testing-and-building)
    * [Adding a New Data Handler](#adding-a-new-data-handler)
      * [Why do I need to add a new data handler?](#why-do-i-need-to-add-a-new-data-handler)
      * [Before you begin](#before-you-begin)
      * [Create a new data handler](#create-a-new-data-handler)
      * [Register the new handler in main.go](#register-the-new-handler-in-maingo)
      * [Test and build your changes](#test-and-build-your-changes)
      * [Adding a new sub resource client](#adding-a-new-sub-resource-client)
    * [Docker Containers](#docker-containers)

## Testing and Building
The best way to build and test your changes is to use the `docker_make` script.
The script downloads all the dependencies, runs the linter and the unit tests 
and builds `kube-volume-controller` in a docker container. Example output is
shown below:

```
$ ./docker_make dep-ensure
dep ensure

$ ./docker_make code-generation
./hack/update-codegen.sh
/go/src/github.com/NervanaSystems/kube-volume-controller/vendor/k8s.io/code-generator /go/src/github.com/NervanaSystems/kube-volume-controller
Note: checking out 'kubernetes-1.9.2'.

You are in 'detached HEAD' state. You can look around, make experimental
changes and commit them, and you can discard any commits you make in this
state without impacting any branches by performing another checkout.

If you want to create a new branch to retain commits you create, you may
do so (now or later) by using -b with the checkout command again. Example:

  git checkout -b <new-branch-name>

HEAD is now at 91d3f6a... Merge pull request #57767 from mbohlool/automated-cherry-pick-of-#57735-upstream-release-1.9
/go/src/github.com/NervanaSystems/kube-volume-controller
Generating deepcopy funcs
Generating clientset for cr:v1 at github.com/nervanasystems/kubevolumecontroller/pkg/client/clientset
Generating listers for cr:v1 at github.com/nervanasystems/kubevolumecontroller/pkg/client/listers
Generating informers for cr:v1 at github.com/nervanasystems/kubevolumecontroller/pkg/client/informers

$ ./docker_make lint
gometalinter --config=./lint.json --vendor .
# Disabling golint for apis since it conflicts with the deepcopy-gen
# annotations.
gometalinter --config=./lint.json --disable=golint ./pkg/apis/...
gometalinter --config=./lint.json ./pkg/hooks/...


$ ./docker_make test
go test --cover ./...
?   	github.com/NervanaSystems/kube-volume-controller	[no test files]
ok  	github.com/NervanaSystems/kube-volume-controller/pkg/apis/cr/v1	0.166s	coverage: 53.4% of statements
?   	github.com/NervanaSystems/kube-volume-controller/pkg/hooks	[no test files]
# go test --cover .
# go test --cover ./pkg/apis/...
# go test --cover ./pkg/hooks/...

$ ./docker_make build
dep ensure
/go/bin/deepcopy-gen --output-base=/go/src --input-dirs=github.com/NervanaSystems/kube-volume-controller/pkg/apis/cr/v1/...
go build -gcflags "-N -l" github.com/NervanaSystems/kube-volume-controller
```

## Adding a New Data Handler

### Why do I need to add a new data handler?

Read the [architecture document][arch-doc-why-dh] for more information on why
adding a new data handler might be necessary.

### Before you begin 

New data handlers can be added by implementing the `DataHandler` interface in 
[handlers.go][handler-interface]. An example implementation for handling data
from S3 can be found in [s3_handler.go][s3-handler]. The following steps 
can be used as reference to add a a new data handler.

### Create a new data handler

Create a new file named `<insert-data-source-type>_handler.go` in
`pkg/handlers/`. Edit the file and add the following code snippet after 
replacing all the comments within `<>` with the appropriate value. 

```go
package handlers

import (
    crv1 "github.com/NervanaSystems/kube-volume-controller/pkg/apis/cr/v1"
)

const (
    <insert-data-source-name>SourceType  crv1.DataSourceType = "<insert-data-source-name>"
)

type <insert-data-source-name>Handler struct {
    <insert-struct-members>
}

func New<insert-data-source-name>Handler(<fill-with-required-parameters>) DataHandler {
    return &<insert-data-source-name>Handler{
        <insert-struct-members-and-values>
    }
}

func (h *<insert-data-source-name>Handler) GetSourceType() crv1.DataSourceType {
    return h.sourceType
}

func (h *<insert-data-source-name>Handler) OnAdd(ns string, vc crv1.VolumeConfig, controllerRef metav1.OwnerReference) crv1.VolumeClaim {
    <insert-code-for-on-add>

    return crv1.VolumeClaim{
        ID:       vc.ID,
        Message:  crv1.SuccessfulVolumeClaimMessage,
    }
}

func (h *<insert-data-source-name>Handler) OnDelete(ns string, vc crv1.VolumeConfig, controllerRef metav1.OwnerReference) {
    <insert-code-for-on-delete>
}
```

### Register the new handler in `main.go`

Replace all the comments within `<>` with appropriate values and the following
code snippet in [main.go][main-file]. If you need additional clients apart from 
the ones already present, please refer to [Adding a new client](#adding-a-new-client) section below.

```go
Handlers := []handlers.DataHandler{
    handlers.NewS3Handler(k8sClientset, resourceClients),
    <insert-new-data-handler-constructor-call>,
}
```

### Test and build your changes

Run `./docker_make build` from the root directory. 

### Adding a new sub resource client

[Dynamic client][dyn-client] is used to create and use the clients required to handle kubernetes resources.
For example, to create a pod client the steps below should be followed:
1. Create the [APIResource][apiresource]: 
```go
podAPIResource := &metav1.APIResource{
		Kind:       "Pod",
		Name:       "pods",
		Group:      "v1",
		Namespaced: true,
	}
``` 
2. Create the dynamic client:
```go
	config.GroupVersion = &corev1.SchemeGroupVersion
	dynClient, err := dynamic.NewClient(config)
```
Note: The same dynamic client can be used if clients for other resources belonging to the same 
group version are to be created.

3. Create a scheme to help the conversion of an [unstructured object][unstructured] to the typed object.
```go
corev1Scheme.AddKnownTypes(corev1.SchemeGroupVersion, &corev1.Pod{})
```

4. Create the client:
```go
    dynClient.Resource(nodeAPIResource, *namespace)
```

Note: See [main.go][main-file] for examples on how to create clients for pod, nodes, pv and pvc using a [dynamic client][dyn-client].

## Docker Containers

The [docker](../docker) directory containers dockerfiles for:

* aws-cli: aws cli tools with a wrapper to support minio
* golang: the build container used by `docker_make`

To build and push the containers to docker hub:
```
cd docker
./build
./push
```

[main-file]: ../main.go
[handler-interface]: ../pkg/handlers/handlers.go
[s3-handler]: ../pkg/handlers/s3_handler.go
[arch-doc-why-dh]: arch.md#relationship-between-volume-and-data
[dyn-client]: https://github.com/kubernetes/client-go/tree/master/dynamic
[apiresource]: https://github.com/kubernetes/apimachinery/blob/master/pkg/apis/meta/v1/types.go#L825
[unstructured]: https://github.com/kubernetes/apimachinery/blob/master/pkg/apis/meta/v1/unstructured/unstructured.go#L41