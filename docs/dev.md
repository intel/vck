# Developer Manual: Volume Controller for Kubernetes (VCK)
  
  * [Developer Manual: Volume Controller for Kubernetes (VCK)](#developer-manual-volume-controller-for-kubernetes-vck)
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

There are several ways to modify `VCK` and test your changes.

### Using "docker_make" script
This method is preferred for developers who have `docker` setup and running on their workstation and don't
want to install `Go` and modify `environment` variables for it.

The `docker_make` script downloads all the dependencies, runs the linter and the unit tests
and builds `kube-volume-controller` in a docker container. Example output is shown below:

```
$ ./docker_make dep-ensure
dep ensure

$ ./docker_make code-generation
./hack/update-codegen.sh
/go/src/github.com/IntelAI/vck/vendor/k8s.io/code-generator /go/src/github.com/IntelAI/vck
Note: checking out 'kubernetes-1.9.2'.

You are in 'detached HEAD' state. You can look around, make experimental
changes and commit them, and you can discard any commits you make in this
state without impacting any branches by performing another checkout.

If you want to create a new branch to retain commits you create, you may
do so (now or later) by using -b with the checkout command again. Example:

  git checkout -b <new-branch-name>

HEAD is now at 91d3f6a... Merge pull request #57767 from mbohlool/automated-cherry-pick-of-#57735-upstream-release-1.9
/go/src/github.com/IntelAI/vck
Generating deepcopy funcs
Generating clientset for cr:v1 at github.com/IntelAI/vck/pkg/client/clientset
Generating listers for cr:v1 at github.com/IntelAI/vck/pkg/client/listers
Generating informers for cr:v1 at github.com/IntelAI/vck/pkg/client/informers

$ ./docker_make lint
gometalinter --config=./lint.json --vendor .
# Disabling golint for apis since it conflicts with the deepcopy-gen
# annotations.
gometalinter --config=./lint.json --disable=golint ./pkg/apis/...
gometalinter --config=./lint.json ./pkg/hooks/...


$ ./docker_make test
go test --cover ./...
?   	github.com/IntelAI/vck	[no test files]
ok  	github.com/IntelAI/vck/pkg/apis/cr/v1	0.166s	coverage: 53.4% of statements
?   	github.com/IntelAI/vck/pkg/hooks	[no test files]
# go test --cover .
# go test --cover ./pkg/apis/...
# go test --cover ./pkg/hooks/...

$ ./docker_make build
dep ensure
/go/bin/deepcopy-gen --output-base=/go/src --input-dirs=github.com/IntelAI/vck/pkg/apis/cr/v1/...
go build -gcflags "-N -l" github.com/IntelAI/vck
```
### Developing on a workstation
This is the preferred method for developers who don't want to run `docker` locally and or don't mind setting up
`Go` development environment on their workstation.

Prior to using this method please ensure you have `go 1.9.2` (or better) development environment setup.
Also make sure `GOROOT`, `GOPATH` and `PATH` environment variables are set to their appropriate values.

For example on `CentOS 7 or Ubuntu 16` add the following lines to your `.bashrc`
```bash
export GOROOT="/usr/lib/go-1.9"
export GOPATH="$HOME/go"
export PATH="$PATH:$GOROOT/bin:$GOPATH/bin"
```

and for `MacOSX` add these lines to your `.bashrc`:
 ```bash
export GOROOT="/usr/local/opt/go/libexec"
export GOPATH=$HOME/go
export PATH=$PATH:$GOROOT/bin:$GOPATH/bin
```
in both cases, install `mercurial` and `socat` either using `yum` or `apt-get` or `brew`.
This is because some dependencies live in `bitbucket` repositories.

Finally make sure `VCK` is present at:
```$GOPATH/src/github.com/IntelAI/vck```

Now you are ready to make changes and test `VCK` as follows:

```
$ make prereq
go get -u \
	github.com/golang/dep/cmd/dep \
	github.com/alecthomas/gometalinter \
	github.com/kubernetes/gengo/examples/deepcopy-gen
gometalinter --install
Installing:
  deadcode
  dupl
  ...
  unused
  varcheck
$ make dep-ensure
dep ensure
$ make code-generation
  ./hack/update-codegen.sh
  ~/go/src/github.com/IntelAI/vck/vendor/k8s.io/code-generator ~/go/src/github.com/IntelAI/vck
  Note: checking out 'kubernetes-1.9.2'.
  ...
  Generating clientset for vck:v1 at github.com/IntelAI/vck/pkg/client/clientset
  Generating listers for vck:v1 at github.com/IntelAI/vck/pkg/client/listers
  Generating informers for vck:v1 at github.com/IntelAI/vck/pkg/client/informers
$ make lint
gometalinter --config=./lint.json --vendor .
# Disabling golint for apis since it conflicts with the deepcopy-gen
# annotations.
gometalinter --config=./lint.json --disable=golint ./pkg/apis/...
gometalinter --config=./lint.json ./pkg/hooks/...
gometalinter --config=./lint.json ./pkg/controller/...
gometalinter --config=./lint.json ./pkg/handlers/...
```

And many more options provided by `Makefile`.

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
    crv1 "github.com/IntelAI/vck/pkg/apis/cr/v1"
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

## Adding Additional Validation
There could be cases in which additional validation is neccessary. Choose the respective method below that corresponds with your needs:


1. [OpenAPI v3 schema] - This is the easiest method but has little flexibility. By modifying `kube-volume-controller-crd.yml`, required fields for all sourceTypes and required types for fields can be set.

Example:

Suppose we have the following VolumeManager
```
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: volumemanagers.vck.intelai.org
spec:
  group: vck.intelai.org
  names:
    kind: VolumeManager
    listKind: VolumeManagerList
    plural: volumemanagers
    singular: volumemanager
  scope: Namespaced
  version: v1alpha1
```

In order to add general validation rules we can add the following:
```
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: volumemanagers.vck.intelai.org
spec:
  group: vck.intelai.org
  names:
    kind: VolumeManager
    listKind: VolumeManagerList
    plural: volumemanagers
    singular: volumemanager
  scope: Namespaced
  version: v1alpha1
  # We add validation here
  validation:
    openAPIV3Schema:
      required:
      - apiVersion
      - kind
      properties:
        apiVersion:
          type: string
          enum:
          - "vck.intelai.org/v1alpha1"
        kind:
          enum:
          - "VolumeManager"
        metadata:
          required:
          - name
          properties:
            name:
              type: string
```
In this case we are requiring certain properties like apiVersion and kind while ensuring properties like apiVersion use type for format or enum for specific required values.

2. [Validation Webhook] - This is a bit more complicated but has significantly more flexibility. By modifying `validation-webhook.go`, any validation rule can be specified in the `validateVolumeManager` and `validate<Source Type>` functions.

Example:

Suppose we want to add a new rule requiring dataPath which is an option specific to only S3. We would add the following to the `validateS3` function:
```
func validateS3(vc vckv1alpha1.VolumeConfig) string {
	...
	if _, ok := vc.Options["dataPath"]; !ok {
		errs = append(errs, "dataPath has to be set in options.")
	}
	...
	return strings.Join(errs, " ")
}
```


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
[OpenAPI v3 schema]: https://github.com/OAI/OpenAPI-Specification/blob/master/versions/3.0.0.md#schemaObject
[Validation Webhook]: https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/#admission-webhooks
