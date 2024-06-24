# User Manual: Volume Controller for Kubernetes (VCK)
  * [User Manual: Volume Controller for Kubernetes (VCK)](#user-manual-volume-controller-for-kubernetes-vck)
    * [Prerequisites](#prerequisites)
    * [Before You Begin](#before-you-begin)
    * [Create a Secret with your AWS Credentials](#create-a-secret-with-your-aws-credentials)
    * [Create a Volume Manager Custom Resource](#create-a-volume-manager-custom-resource)
    * [Create a Pod using the Custom Resource Status](#create-a-pod-using-the-custom-resource-status)
    * [Create a Deployment using the Custom Resource Status](#create-a-deployment-using-the-custom-resource-status)
    * [Create a Deployment using the VCK Initializer](#create-a-deployment-using-the-vck-initializer)
    * [Types of Sources](#types-of-sources)
    * [Data distribution](#data-distribution)

## Prerequisites

- Kubernetes v1.9+ with [`VolumeScheduling`][vol-sched] feature gate enabled
- [Kubectl][kubectl]
- [Helm][helm]
- [VCK deployed on the cluster][ops-doc]

## Before You Begin

Check if the cluster has the volume manager CRD. Example command to verify
this is shown below.

```sh
$ kubectl get crd
NAME                            AGE
volumemanagers.vck.intelai.org   1h

$ kubectl get volumemanagers
No resources found.
```

Also check if the Controller is installed in your namespace:
```sh
$ helm list
NAME            	REVISION	UPDATED                 	STATUS  	CHART                        	NAMESPACE
vck-ashahba     	1       	Thu Apr 26 13:10:36 2018	DEPLOYED	kube-volume-controller-v0.1.0	ashahba
```

If not follow the instructions in the [operator manual][ops-doc] first.

Set your current context to use the same namespace as the controller.
The following commands can be used after editing the comments within `<>`.

```sh
$ kubectl config set-context $(kubectl config current-context) --namespace=<insert-namespace-here>
```

## Create a Secret with your AWS Credentials
_Note: This secret is only required when using the `S3` and `S3-Dev` data source type._

Use the commands below to create a secret with your AWS credentials. Before
using the command below, make sure to replace the comments within `<>` with
appropriate values.

```sh
$ export AWS_ACCESS_KEY_ID="<insert-your-aws-access-key-id>"
$ export AWS_SECRET_ACCESS_KEY="<insert-your-aws-secret-access-key>"
$ kubectl create secret generic aws-creds --from-literal=awsAccessKeyID=${AWS_ACCESS_KEY_ID} --from-literal=awsSecretAccessKey=${AWS_SECRET_ACCESS_KEY}
```

## Create a Volume Manager Custom Resource

Using the [example custom resource manifest][cr-example], create a custom
resource. Example commands are shown below. Before using the command below,
make sure to replace the comments within `<>` with appropriate values.
For the `S3` or `S3-Dev` data source types, the value of `spec.options.awsCredentialsSecretName`
in the custom resource should be set to the secret name created using the
instructions [above](#create-a-secret-with-your-aws-credentials).

```sh
$ kubectl create -f resources/customresources/s3/one-vc.yaml
volumemanager "vck-example" created

$ kubectl get volumemanager vck-example -o yaml
apiVersion: vck.intelai.org/v1
kind: VolumeManager
metadata:
  clusterName: ""
  creationTimestamp: 2018-02-21T20:22:30Z
  generation: 0
  name: vck-example
  namespace: vck-testing
  resourceVersion: "4722186"
  selfLink: /apis/vck.intelai.org/v1/namespaces/vck-testing/volumemanagers/vck-example
  uid: f0e352bd-1744-11e8-9cc4-42010a8a026b
spec:
  state: ""
  volumeConfigs:
  - accessMode: ReadWriteOnce
    capacity: 5Gi
    id: vol1
    labels:
      key1: val1
      key2: val2
    options:
      awsCredentialsSecretName: aws-creds
    replicas: 1
    sourceType: S3
    sourceURL: s3://neon-stockdatasets/cifar-100-python.tar.gz
status:
  message: successfully deployed all sub-resources
  state: Running
  volumes:
  - id: vol1
    message: success
    nodeAffinity:
      requiredDuringSchedulingIgnoredDuringExecution:
        nodeSelectorTerms:
        - matchExpressions:
          - key: kubernetes.io/hostname
            operator: In
            values:
            - cluster-node-1
    volumeSource:
      hostPath:
        path: /var/datasets/vck-resource-f0e5a3ba-1744-11e8-a808-0a580a44065b
```

Other examples on custom resource manifest can be found in [resources][resources-dir]
directory. For details about source types and their fields, refer to [types of sources](#types-of-sources).

## Create a Pod using the Custom Resource Status

Using the [example pod manifest][pod-example], create a pod.
Example commands are shown below. Before using the command below, make sure to
replace the comments within `<>` with appropriate values of node affinity and
the volume source from the CR status. Here is an example of an edited spec for a S3
data source type:

```yaml
  affinity:
    nodeAffinity:
      requiredDuringSchedulingIgnoredDuringExecution:
        nodeSelectorTerms:
        - matchExpressions:
          - key: kubernetes.io/hostname
            operator: In
            values:
            - cluster-node-1
  volumes:
    - name: dataset-claim
      hostPath:
        path: /var/datasets/vck-resource-f0e5a3ba-1744-11e8-a808-0a580a44065b
```

Depending upon the source type, node
affinity might not be required. [Types of sources](#types-of-sources) provides
more details on how to edit the field(s) in the pod template spec before using the
command below.

```sh
$ kubectl create -f resources/pods/vck-pod.yaml
pod "vck-claim-pod" created
```

## Create a Deployment using the Custom Resource Status

Using the [example deployment manifest][dep-example], create a deployment.
Example commands are shown below. Before using the command below, make sure to
replace the comments within `<>` with appropriate values of node affinity and
the volume source from the CR status. Depending upon the source type, node
affinity might not be required. [Types of sources][#types-of-sources] provides
details on which field(s) need to filled in the deployment template before using the
command below.

```sh
$ kubectl create -f resources/deployments/vck-deployment.yaml
deployment "vck-example-deployment" created
```

## Create a Deployment using the VCK Initializer

The VCK Initializer will ensure the  volume manager data is only injected into Deployments with an `initializer.kubernetes.io/vck` annotation set to a non-empty value.

```yaml

"initializer.kubernetes.io/vck": '{
        "name": "<insert-your-vck-name>",
        "id": "<insert-your-vck-id>",
        "containers": [
          {
            "name": "<insert-your contianer-name>",
            "mount-path" : "<insert-your-mount-path>"
          }
        ],
      }'
```

| Key                  | Required | Description                                    | Default      |
|:----------------------|:---------:|:------------------------------------------------|:--------------|
| name                 | yes      | The VCK name to append volumes to containers   |              |
| id                   | no       | The id of the volume to append to container           | first volume |
| containers           | no       | Name and MountPath of container                | all          |
| container.name       | yes      | Name of the container to append the VCK volume |              |
| container.mount-path | no       | Path for the VCK to mount the volume           | /var/dataset |

The id key is optional it picks the first volume by default, similarly the container object is optional it picks all containers by default and appends it to default mount path "/var/datasets". If the container object just contains the name,vck is appended to default mount path  "/var/datasets".

* Annotation with only name

```yaml

"initializer.kubernetes.io/vck": '{
        "name": "<insert-your-vck-name>"
      }'
```

* Annotation with no containers

```yaml
"initializer.kubernetes.io/vck": '{
        "name": "<insert-your-vck-name>",
        "id": "<insert-your-vck-id>"
      }'
```

* Annotation with no container.mount-path

```yaml
"initializer.kubernetes.io/vck": '{
        "name": "<insert-your-vck-name>",
        "id": "<insert-your-vck-id>",
        "containers": [
          {
            "name": "<insert-your contianer-name>"
          }
        ],
      }'
```

## Types of Sources

The following source types are currently implemented:
* S3: Files present in an S3 bucket and provided as `volumeConfig.sourceURL` in the CR are downloaded/synced onto the number of nodes equal to `volumeConfig.replicas` and made available as a hostPath volume. Node affinity details are provided through `volume.nodeAffinity` to guide the scheduling of pods.
* NFS: The path exported by an NFS server is mounted and made available as a PVC.
* Pachyderm: The repo, branch and file in [Pachyderm][pachyderm] and provided as `volumeConfig.options["repo"]`, `volumeConfig.options["branch"]` and `volumeConfig.options["filePath"]` in the CR are downloaded/synced onto the number of nodes equal to `volumeConfig.replicas` and made available as a hostPath volume. Node affinity details are provided through `volume.nodeAffinity` to guide the scheduling of pods.

_NOTES:
For minio configure the setting `volumeConfig.endpointURL` to point to your minio service url.
When the CR for S3 source type is deleted, all the replicated data is also deleted. Care should be taken when deleting CRs as objects, such as pods, using the CR will lose the data._

For examples on how to define and use the different types, please refer to the examples in [resources][resources-dir].

A brief description of each source type is provided below.

| Type         | Fields | Required                         |  Description                                          | Supported Access Modes | Field(s) provided in CR status |
|:-------------|:----------------------------------------|:----|:--------------------------------------------------|:-----------------------|:-------------------------------|
| `S3`         | `volumeConfig.options["sourceURL"]`     | Yes | The s3 url to download the data from. End the sourceURL with a `/` to recursively copy | `ReadWriteOnce`        | `volumeSource`                 |
|              | `volumeConfig.options["endpointURL"]`   | No | The s3 compatible service endpoint (i.e. minio url).  Defaults to "https://s3.amazonaws.com"          |                        | |
|              | `volumeConfig.replicas`                 | Yes | The number of nodes this data should be replicated on. |                        | `nodeAffinity`                 |
|              | `volumeConfig.options["dataPath"]`                 | No | The  data path on the node where s3 data would be downloaded.  Defaults to "/var/datasets" |                        | `volumeSource`                 |
|              | `volumeConfig.options["awsCredentialsSecretName]` | Yes | The name of the secret with AWS credentials to access the s3 data              |                        | |
|              | `volumeConfig.options["timeoutForDataDownload"]`  | No | The timeout for download of s3 data. Defaults to 5 minutes. [[Format]](https://golang.org/pkg/time/#ParseDuration) |                        | |
|              | `volumeConfig.options["distributionStrategy"]`    | No | The [distribution strategy](#data-distribution) to use to distribute the data across the replicas |                        | |
|              | `volumeConfig.options["resync"]`    | No | The `resync` option syncs back the changes made in the local directory to the source. Please read through the [notes](#resync) before using this option. |                        | |
| `NFS`        | `volumeConfig.options["server"]`        | Yes | Address of the NFS server.                             |`ReadWriteMany`         | `volumeSource`                 |
|              | `volumeConfig.options["path"]`          | Yes | The path exported by the NFS server.                   |`ReadOnlyMany`          | |
|              | `volumeConfig.accessMode     `          | Yes | Access mode for the volume config.                     |                        | |
| `Pachyderm`  | `volumeConfig.options["repo"]`          | Yes | Pachyderm repo.                             |`ReadWriteOnce`         | `volumeSource`                 |
|              | `volumeConfig.options["branch"]`        | Yes | Branch of that repo.                   |          | |
|              | `volumeConfig.options["inputPath"]`     | Yes | File path in the branch.                 |          | |
|              | `volumeConfig.options["outputPath"]`    | Yes | Output path for the files.                 |          | |
|              | `volumeConfig.options["pachydermServiceAddress"`]                 | No | The address and port of the pachyderm service. Defaults to "pachd.default.svc:650". |                        |                  |
|              | `volumeConfig.replicas`                 | Yes | The number of nodes this data should be replicated on. |                        | `nodeAffinity`                 |
|              | `volumeConfig.options["timeoutForDataDownload"]`  | No | The timeout for download of data. Defaults to 5 minutes. [[Format]](https://golang.org/pkg/time/#ParseDuration) |                        | |
|              | `volumeConfig.accessMode     `          | Yes | Access mode for the volume config.                     |                        | |

Status of the CR provides information on the volume source and node affinity.
Example status fields for the different source types and a description on
what needs to be changed in the [pod template][pod-example] to use these
source types is given below.

* S3:
  ```yaml
  - id: vol1
    message: success
    nodeAffinity:
      requiredDuringSchedulingIgnoredDuringExecution:
        nodeSelectorTerms:
        - matchExpressions:
          - key: kubernetes.io/hostname
            operator: In
            values:
            - cluster-node-1
            - cluster-node-2
    volumeSource:
      hostPath:
        path: /var/datasets/vck-resource-a2140d72-11c2-11e8-8397-0a580a440340
  ```
  The [node affinity][node-affinity] above can be used as-is in a pod spec
  along with the host path above as a volume to access the s3 data.
  More specifically, the snippets below from the CR status above needs to
  inserted in the [volumes field][pod-example-vol] and [affinity field][pod-example-aff]
  of the example [pod template][pod-example], respectively, in order to use it with the pod.

  ```yaml
      hostPath:
        path: /var/datasets/vck-resource-a2140d72-11c2-11e8-8397-0a580a440340
  ```

  ```yaml
    nodeAffinity:
      requiredDuringSchedulingIgnoredDuringExecution:
        nodeSelectorTerms:
        - matchExpressions:
          - key: kubernetes.io/hostname
            operator: In
            values:
            - cluster-node-1
            - cluster-node-2
  ```

* NFS
  ```yaml
  - id: vol2
        message: success
        nodeAffinity: {}
        volumeSource:
          persistentVolumeClaim:
            claimName: vck-resource-a216ed4a-11c2-11e8-8397-0a580a440340
  ```
  The claim can be used in a pod to access the data.
  More specifically, the snippet below from the CR status above needs to inserted in the
  [volumes field][pod-example-vol] of the example [pod template][pod-example]
  in order to use it with the pod.

  ```yaml
      persistentVolumeClaim:
        claimName: vck-resource-a150fd63-11c4-11e8-8397-0a580a440340
  ```
  ### Caveats ###
    The NFS server ip and path are not validated, so please ensure that the servers are routable and paths are valid prior to the creation of the VolumeManager CR.
    In case an invalid `server` or `path` is used, Kubernetes publishes an event similar to the following during first attempt to use the PVC:
    ```
    Unable to mount volumes for pod : timeout expired waiting for volumes to attach/mount for pod ...
    ```

* Pachyderm:
  ```yaml
  - id: vol1
    message: success
    nodeAffinity:
      requiredDuringSchedulingIgnoredDuringExecution:
        nodeSelectorTerms:
        - matchExpressions:
          - key: kubernetes.io/hostname
            operator: In
            values:
            - cluster-node-1
            - cluster-node-2
    volumeSource:
      hostPath:
        path: /var/datasets/vck-resource-a2140d72-11c2-11e8-8397-0a580a440340
  ```
  The [node affinity][node-affinity] above can be used as-is in a pod spec
  along with the host path above as a volume to access the pachyderm data.
  More specifically, the snippets below from the CR status above needs to
  inserted in the [volumes field][pod-example-vol] and [affinity field][pod-example-aff]
  of the example [pod template][pod-example], respectively, in order to use it with the pod.

  ```yaml
      hostPath:
        path: /var/datasets/vck-resource-a2140d72-11c2-11e8-8397-0a580a440340
  ```

  ```yaml
    nodeAffinity:
      requiredDuringSchedulingIgnoredDuringExecution:
        nodeSelectorTerms:
        - matchExpressions:
          - key: kubernetes.io/hostname
            operator: In
            values:
            - cluster-node-1
            - cluster-node-2
  ```



To add a new source type, a new handler specific to the source type is required. Please refer to the [developer manual][dev-doc] for more details.

## Data distribution

For the S3 source type the user can provide a distribution strategy which should be of the form: `{"glob_pattern_1": #replicas, "glob_pattern_2": #replicas, ...}`. [Glob][glob] patterns are supported in this case and the total number of replicas across all the patterns
should equal the number or replicas in the spec.
The strategy specifies that the files found in the specified source by applying the given glob pattern will be replicated across #replicas nodes, For example for the given yaml:
```yaml
apiVersion: vck.intelai.org/v1
kind: VolumeManager
metadata:
  name: vck-example1
  namespace: <insert-namespace-here>
spec:
  volumeConfigs:
    - id: "vol1"
      replicas: 4
      sourceType: "S3"
      accessMode: "ReadWriteOnce"
      capacity: 5Gi
      labels:
        key1: val1
        key2: val2
      options:
        awsCredentialsSecretName: aws-secret
        sourceURL: "s3://foo/bar"
        distributionStrategy: '{"*0_1*": 2, "*1_1*": 2}'
        # dataPath: <insert-data-path-here-optional>"
```

all files matching the pattern `*0_1*` in the bucket `s3://foo/bar` would be synced in 2 replicas and all files matching `*1_1*` would be synced in the remaining 2 replicas.

## Resync

For the S3 source type, the user can opt-in to resync the contents of the local directory with the source (i.e.,
the S3 object soruce). VCK watches for any changes made to the local directory and syncs with the remote S3 object
store when this option is enabled. When `resync` is set for the S3 source type, only one replica is supported. 
Note that the files are overwritten in the remote S3 object store. 


[ops-doc]: ops.md
[dev-doc]: dev.md
[arch-doc]: arch.md
[resources-dir]: ../resources/customresources
[vol-sched]: https://github.com/kubernetes/features/issues/490
[node-affinity]: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/#node-affinity-beta-feature
[helm]: https://docs.helm.sh/using_helm/
[kubectl]: https://kubernetes.io/docs/tasks/tools/install-kubectl/
[cr-example]: ../resources/customresources/s3/one-vc.yaml
[pod-example]: ../resources/pods/vck-pod.yaml
[pod-example-vol]: ../resources/pods/vck-pod.yaml#L10
[pod-example-aff]: ../resources/pods/vck-pod.yaml#L7
[dep-example]: ../resources/deployments/vck-deployment.yaml
[secret-example]: ../resources/secrets/aws-secret.yaml
[secret-encoding]: https://kubernetes.io/docs/concepts/configuration/secret/#creating-a-secret-manually
[pachyderm]: http://pachyderm.io
[glob]: https://en.wikipedia.org/wiki/Glob_(programming)
