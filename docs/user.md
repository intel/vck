# User Manual: Kubernetes Volume Controller (KVC)
  * [User Manual: Kubernetes Volume Controller (KVC)](#user-manual-kubernetes-volume-controller-kvc)
    * [Prerequisites](#prerequisites)
    * [Before You Begin](#before-you-begin)
    * [Create a Secret with your AWS Credentials](#create-a-secret-with-your-aws-credentials)
    * [Create a Volume Manager Custom Resource](#create-a-volume-manager-custom-resource)
    * [Create a Pod using the Custom Resource Status](#create-a-pod-using-the-custom-resource-status)
    * [Create a Deployment using the Custom Resource Status](#create-a-deployment-using-the-custom-resource-status)
    * [Types of Sources](#types-of-sources)

## Prerequisites

- Kubernetes v1.9+ with [`VolumeScheduling`][vol-sched] feature gate enabled
- [Kubectl][kubectl]
- [Helm][helm]
- [KVC deployed on the cluster][ops-doc]

## Before You Begin

Check if the cluster has the volume manager CRD. Example command to verify 
this is shown below.

```sh
$ kubectl get crd
NAME                            AGE
volumemanagers.kvc.kubeflow.org   1h

$ kubectl get volumemanagers
No resources found.
```

Also check if the Controller is installed in your namespace:
```
$ helm list
NAME            	REVISION	UPDATED                 	STATUS  	CHART                        	NAMESPACE
kvc-ashahba     	1       	Thu Apr 26 13:10:36 2018	DEPLOYED	kube-volume-controller-v0.1.0	ashahba
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
volumemanager "kvc-example" created

$ kubectl get volumemanager kvc-example -o yaml
apiVersion: kvc.kubeflow.org/v1
kind: VolumeManager
metadata:
  clusterName: ""
  creationTimestamp: 2018-02-21T20:22:30Z
  generation: 0
  name: kvc-example
  namespace: kvc-testing
  resourceVersion: "4722186"
  selfLink: /apis/kvc.kubeflow.org/v1/namespaces/kvc-testing/volumemanagers/kvc-example
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
        path: /var/datasets/kvc-resource-f0e5a3ba-1744-11e8-a808-0a580a44065b
```

Other examples on custom resource manifest can be found in [resources][resources-dir]
directory. For details about source types and their fields, refer [types of sources](#types-of-sources).

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
        path: /var/datasets/kvc-resource-f0e5a3ba-1744-11e8-a808-0a580a44065b
```

Depending upon the source type, node
affinity might not be required. [Types of sources](#types-of-sources) provides
more details on how to edit the field(s) in the pod template spec before using the
command below.

```sh
$ kubectl create -f resources/pods/kvc-pod.yaml
pod "kvc-claim-pod" created
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
$ kubectl create -f resources/deployments/kvc-deployment.yaml
deployment "kvc-example-deployment" created
```

## Types of Sources
The following source types are currently implemented:
* S3-Dev: Files present in an S3 bucket and provided as `volumeConfig.sourceURL` in the CR are downloaded/synced and made available as a PVC. Only 1 replica is allowed. This source type should only be used for development and testing purposes.
* S3: Files present in an S3 bucket and provided as `volumeConfig.sourceURL` in the CR are downloaded/synced onto the number of nodes equal to `volumeConfig.replicas` and made available as a hostPath volume. Node affinity details are provided through `volume.nodeAffinity` to guide the scheduling of pods.
* NFS: The path exported by an NFS server is mounted and made available as a PVC.

NOTE: For minio configure the setting `volumeConfig.endpointURL` to point to your minio service url.

For examples on how to define and use the different types, please refer to the examples in [resources][resources-dir].

A brief description of each source type is provided below.

| Type    | Fields | Required                         |  Description                                          | Supported Access Modes | Field(s) provided in CR status | 
|:--------|:----------------------------------------|:----|:--------------------------------------------------|:-----------------------|:-------------------------------|
| `S3-Dev`| `volumeConfig.sourceURL`                | Yes | The s3 url to download the data from. End the sourceURL with a `/` to recursively copy |`ReadWriteOnce`         | `volumeSource`                 |
|         | `volumeConfig.endpointURL`              | No | The s3 compatible service endpoint (i.e. minio url)         |                        | |
|         | `volumeConfig.replicas`                 | No | Field is ignored for this source type.                 |                        | |
|         | `volumeConfig.options["dataPath"]`                 | No | The  data path on the node where s3 data would be downloaded  |                        | `volumeSource`                 |
|         | `volumeConfig.options["awsCredentialsSecretName"]` | Yes | The name of the secret with AWS credentials to access the s3 data              |                        | |
|         | `volumeConfig.options["timeoutForDataDownload"]`  | No | The timeout for download of s3 data. Defaults to 5 minutes. [[Format]](https://golang.org/pkg/time/#ParseDuration) |                        | |
| `S3`    | `volumeConfig.sourceURL`                | Yes | The s3 url to download the data from. End the sourceURL with a `/` to recursively copy | `ReadWriteOnce`        | `volumeSource`                 |
|         | `volumeConfig.endpointURL`              | No | The s3 compatible service endpoint (i.e. minio url)          |                        | |
|         | `volumeConfig.replicas`                 | Yes | The number of nodes this data should be replicated on. |                        | `nodeAffinity`                 |
|         | `volumeConfig.options["dataPath"]`                 | No | The  data path on the node where s3 data would be downloaded |                        | `volumeSource`                 |
|         | `volumeConfig.options["awsCredentialsSecretName]` | Yes | The name of the secret with AWS credentials to access the s3 data              |                        | |
|         | `volumeConfig.options["timeoutForDataDownload"]`  | No | The timeout for download of s3 data. Defaults to 5 minutes. [[Format]](https://golang.org/pkg/time/#ParseDuration) |                        | |
| `NFS`   | `volumeConfig.options["server"]`        | Yes | Address of the NFS server.                             |`ReadWriteMany`         | `volumeSource`                 |
|         | `volumeConfig.options["path"]`          | Yes | The path exported by the NFS server.                   |`ReadOnlyMany`          | |
|         | `volumeConfig.accessMode     `          | Yes | Access mode for the volume config.                     |                        | |

Status of the CR provides information on the volume source and node affinity.
Example status fields for the different source types and a description on 
what needs to be changed in the [pod template][pod-example] to use these
source types is given below.

* S3-Dev:
  ```yaml
    - id: vol1
    message: success
    nodeAffinity: {}
    volumeSource:
      persistentVolumeClaim:
        claimName: kvc-resource-a150fd63-11c4-11e8-8397-0a580a440340
  ```
  The claim can be used in a pod to access the data. More specifically, the
  snippet below from the CR status above needs to inserted in the 
  [volumes field][pod-example-vol] of the example [pod template][pod-example]
  in order to use it with the pod.

  ```yaml
      persistentVolumeClaim:
        claimName: kvc-resource-a150fd63-11c4-11e8-8397-0a580a440340
  ```

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
        path: /var/datasets/kvc-resource-a2140d72-11c2-11e8-8397-0a580a440340
  ```
  The [node affinity][node-affinity] above can be used as-is in a pod spec
  along with the host path above as a volume to access the s3 data.
  More specifically, the snippets below from the CR status above needs to
  inserted in the [volumes field][pod-example-vol] and [affinity field][pod-example-aff]
  of the example [pod template][pod-example], respectively, in order to use it with the pod.
      
  ```yaml
      hostPath:
        path: /var/datasets/kvc-resource-a2140d72-11c2-11e8-8397-0a580a440340
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
            claimName: kvc-resource-a216ed4a-11c2-11e8-8397-0a580a440340
  ```
  The claim can be used in a pod to access the data.
  More specifically, the snippet below from the CR status above needs to inserted in the 
  [volumes field][pod-example-vol] of the example [pod template][pod-example]
  in order to use it with the pod.
  
  ```yaml
      persistentVolumeClaim:
        claimName: kvc-resource-a150fd63-11c4-11e8-8397-0a580a440340
  ```

To add a new source type, a new handler specific to the source type is required. Please refer to the [developer manual][dev-doc] for more details.

[ops-doc]: ops.md
[dev-doc]: dev.md
[arch-doc]: arch.md
[resources-dir]: ../resources/customresources
[vol-sched]: https://github.com/kubernetes/features/issues/490
[node-affinity]: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/#node-affinity-beta-feature
[helm]: https://docs.helm.sh/using_helm/
[kubectl]: https://kubernetes.io/docs/tasks/tools/install-kubectl/
[cr-example]: ../resources/customresources/s3/one-vc.yaml
[pod-example]: ../resources/pods/kvc-pod.yaml
[pod-example-vol]: ../resources/pods/kvc-pod.yaml#L10
[pod-example-aff]: ../resources/pods/kvc-pod.yaml#L7
[dep-example]: ../resources/deployments/kvc-deployment.yaml
[secret-example]: ../resources/secrets/aws-secret.yaml
[secret-encoding]: https://kubernetes.io/docs/concepts/configuration/secret/#creating-a-secret-manually 
