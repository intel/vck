# User Manual: Kubernetes Volume Controller (KVC)
  * [User Manual: Kubernetes Volume Controller (KVC)](#user-manual-kubernetes-volume-controller-kvc)
    * [Prerequisites](#prerequisites)
    * [Before You Begin](#before-you-begin)
    * [Create a Volume Manager Custom Resource](#create-a-volume-manager-custom-resource)
    * [Create a Pod using the PVC as a Volume.](#create-a-pod-using-the-pvc-as-a-volume)


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
volumemanagers.aipg.intel.com   1h

$ kubectl get volumemanagers
No resources found.
```

If not follow the instructions in the [operator manual][ops-doc] first.

## Create a Volume Manager Custom Resource

Using the [example custom resource manifest][cr-example], create a custom
resource. Example commands are shown below. Before using the command below,
make sure to replace the comments within `<>` with appropriate values.

```sh
$ kubectl create -f resources/customresources/one-sourcetype-one-vc.yaml
volumemanager "kvc-example" created

$ kubectl describe volumemanager kvc-example
Name:         kvc-example
Namespace:    kvc-testing
Labels:       <none>
Annotations:  <none>
API Version:  aipg.intel.com/v1
Kind:         VolumeManager
Metadata:
  Cluster Name:
  Creation Timestamp:  2018-02-03T00:44:45Z
  Generation:          0
  Resource Version:    1174103
  Self Link:           /apis/aipg.intel.com/v1/namespaces/kvc-testing/volumemanagers/kvc-example
  UID:                 6e4e2901-087b-11e8-9cc4-42010a8a026b
Spec:
  State:
  Volume Configs:
    Access Mode:  ReadWriteOnce
    Capacity:     5Gi
    Id:           vol1
    Labels:
      Key 1:  val1
      Key 2:  val2
    Options:
      Aws Access Key:     foobarbazfoobarbazfoobarbaz
      Aws Access Key ID:  FOOBARFOOBAR
    Replicas:             1
    Source Type:          S3
    Source URL:           s3://stockdatasets/cifar-100-python.tar.gz
Status:
  Message:  successfully deployed all sub-resources
  State:    Running
  Volume Claims:
    Id:       vol1
    Message:  success
    Pvc Name:
      kvc-resource-6e514b6b-087b-11e8-82f6-0a580a44052f
Events:  <none>
```

Other examples on custom resource manifest can be found in [resources][resources-dir]
directory. For details about source types and their fields, refer [types of sources](#types-of-sources).

## Create a Pod using the PVC as a Volume. 

Using the [example pod manifest][pod-example], create a custom resource.
Example commands are shown below. Before using the command below, make sure to
replace the comments within `<>` with appropriate values.

```sh
$ kubectl create -f resources/pods/pvc-pod.yaml
pod "kvc-claim-pod" created
```

## Types of sources
The following source types are currently implemented:
* S3: Files present in the bucket and provided as `volumeConfig.sourceURL` in the CR are downloaded/synced and made available as a PVC.
* NFS: The path exported by an NFS server is mounted and made available as a PVC.

For examples on how to define and use the different types, please refer to the examples in [resources][resources-dir].

Each source type requires some extra mandatory files. The description of these fields for each source type is given below:

| Type           | Required Fields                                    |  Description                             | 
|:---------------|:---------------------------------------------------|:-----------------------------------------|
| `S3`           | `volumeConfig.sourceURL`                           | The s3 url to download the data from     |
|                | `volumeConfig.options["awsAccessKeyID]`            | The aws access key to access the s3 data |
|                | `volumeConfig.options["awsAccessKey"]`             | The aws secret key to access the s3 data |
| `NFS`          | `volumeConfig.options["server"]`                   | Address of the NFS server                |
|                | `volumeConfig.options["path"]`                     | The path exported by the NFS server      |

To add a new source type, a new handler specific to the source type is required. Please refer to the [developer manual][dev-doc] for more details.

[ops-doc]: ops.md
[dev-doc]: dev.md
[arch-doc]: arch.md
[resources-dir]: resources/customresources
[vol-sched]: https://github.com/kubernetes/features/issues/490
[helm]: https://docs.helm.sh/using_helm/
[kubectl]: https://kubernetes.io/docs/tasks/tools/install-kubectl/
[cr-example]: ../resources/customresources/s3/one-vc.yaml
[pod-example]: ../resources/pods/pvc-pod.yaml
