# Architecture: Volume Controller for Kubernetes (VCK)

  * [Architecture: Volume Controller for Kubernetes (VCK)](#architecture-volume-controller-for-kubernetes-vck)
    * [Acronyms](#acronyms)
    * [Concepts](#concepts)
    * [Overview](#overview)
      * [Goals](#goals)
      * [Non-Goals](#non-goals)
    * [API Schema](#api-schema)
    * [The VCK Controller](#the-vck-controller)
    * [Relationship Between Volume and Data](#relationship-between-volume-and-data)
    * [Source Type Support Status](#source-type-support-status)

## Acronyms

| Term           | Meaning |
| :------------- | :------ |
| CR | [Custom Resource][cr] |
| CRD | [Custom Resource Definition][crd] |
| NFS | Network File System |
| PV | [Persistent Volume][pv] |
| PVC | [Persistent Volume Claim][pvc] |

## Concepts

| Term           | Meaning |
| :------------- | :------ |
| Controller | The process which drives a Kubernetes object from its _current_ state to the _desired_ state |
| Source Type| The source type for the data being stored (e.g., S3, NFS) |
| `volumemanager` | The CRD `Kind` for VCK |

## Overview

VCK provides basic volume and data management using [volumes][vols] and
[volume sources][volsources] in a Kubernetes cluster. It uses CRDs and
controllers to create the [volumes][vols] and [volume sources][volsources] and perform operations
necessary for the data to be available to users. The user needs to have
interactions only with CRs. The rest of the details are abstracted away by VCK.

### Goals
The end goals of this project are listed below:

- __Data source support:__ VCK should support exposing data from different sources such as S3, NFS and
local disk as volumes.
- __Data distribution:__ VCK should support data replication for distributed job types.
- __Data affinity:__ VCK should enable data affinity and gravity. It should use
existing mechanisms such as [volume scheduling][vol-sched] and [node affinity][node-aff] when possible.
- __Data caching:__ VCK should enable the pre-population of data if required.
- __Data streaming:__ VCK should provide abstraction for streaming data
services. Jobs should be able to start as soon as the first stream or batch of
data is available.
- __Job output:__ VCK should allow output data to be gathered when required.
- __Garbage collection:__ VCK should evict data in case of disk pressure.

### Non-Goals
- VCK does not aim to be a solution to all your volume and data
management problems.
- VCK does not solve any of the shortcomings or drawbacks with Kubernetes. If
there is an issue in Kubernetes, the same issue exists with VCK.

## API Schema

Using VCK we extend the Kubernetes API to include a new CRD called
`volumemanager`. The schema to create a `volumemanager` CR is described
below:

| Field Name                    | Type                                              | definition                                                                                                 |
| :-----------------------------| :-------------------------------------------------| :----------------------------------------------------------------------------------------------------------|
| `apiVersion`*                 | `string`                                          | API version of volume manager                                                                              |
| `kind`*                       | enum: `VolumeManager`                             | Type. Only allowed value is `VolumeManager`                                                                |
| `metadata.name`*              | `string`                                          | Name of the volume manager instance                                                                        |
| `spec.volumes`*               | array of `volumeConfig`                           | Volumes and data information                                                                               |
| `volumeConfig.id`*            | `string`                                          | An identifier for the volume                                                                               |
| `volumeConfig.replicas`*      | `int`                                             | Number of replicas required on distinct compute nodes                                                      |
| `volumeConfig.sourceType`*    | `string`                                          | Source type of the dataset to be used by the volume (e.g., S3, NFS)                                        |
| `volumeConfig.sourceURL`      | `string`                                          | Source URL of the data set                                                                                 |
| `volumeConfig.accessMode`*    | `string`                                          | Type of access mode                                                                                        |
| `volumeConfig.capacity`*      | `string`                                          | Size requested for the volume                                                                              |
| `volumeConfig.labels`*        | `map[string]string`                               | Any labels required for the volume                                                                         |
| `volumeConfig.options`        | `map[string]string`                               | Any options required for the volume                                                                        |
| `volumeConfig.nodeAffinity`   | [NodeAffinity][node-aff]                          | The node affinity to restrict or prefer the data placement                                                 |
| `volumeConfig.tolerations`    | [Tolerations][tolerations]                        | Any tolerations the CR should respect                                                                      |
| `spec.state*`                 | enum: `Pending`, `Running`, `Failed`, `Completed` |  The desired state for this volume manager instance                                                        |
| `status.volumes`              | array of `volume`                                 | volume information                                                                                         |
| `volume.id`                   | `string`                                          | An identifier for the volume. There is a one-to-one mapping between `volumeConfig.id` and `volumeClaim.id` |
| `volume.volumeSource`         | [VolumeSource][volsources]                                    | A volume source associated with the `volume`                                                       |
| `volume.message`              | `string`                                          | A message associated with the state of this `volume`                                                       |
| `volume.nodeAffinity`         | [NodeAffinity][node-aff]                             | A node affinity to guide the pod scheduling for data gravity                                        |
| `status.state`                | enum: `Pending`, `Running`, `Failed`, `Completed` |  The  current state of this volume manager instance                                                         |
| `status.message`              | `string`                                          | A message associated with the current state of this volume manager instance                                |

Fields marked with `*` are mandatory.
## The VCK Controller

The VCK controller uses [volumes][vols], [volume sources][volsources], Pods to manage volumes and the associated
data in Kubernetes. The following are the responsibilities of the controller:

__Data source support:__ The controller will transparently support different
data sources. Some of the data sources such as NFS are natively supported
by PVs.

__Data distribution:__ In case of a shared file system, data distribution will
be handled using access modes in volumes. There are three different types of access
modes:
- ReadWriteOnce – the volume can be mounted as read-write by a single node.
- ReadOnlyMany – the volume can be mounted read-only by many nodes.
- ReadWriteMany – the volume can be mounted as read-write by many nodes.

These access modes can be used as long as the shared file system supports it.

If the data is stored somewhere else (e.g., S3) and it needs to be available in
the source path, the controller is responsible to download the data and
replicate it across `N` number of nodes as specified by the
`volumeConfig.replicas` field in the API schema. Depending on the source type, either
PVs of [local][local-pv-type] volume source type or [hostPath][hostPath] volumes are
created.

__Data affinity:__ When required, data affinity will be transparently supported
using either [volume scheduling][vol-sched] or [node affinity][node-aff] features
in Kubernetes.

__Data caching:__ As long as the backing volume is available, it can be used
in any pod. The controller will be responsible to provide the volume source
and the node affinity associated with a `volume`.

__Data streaming:__ Data services, such as [Aeon][aeon], use a caching mechanism to
provide data streaming services. As an example, if Aeon is used for data
streaming service, it uses a cache to stream the data to a compute node. The
location and size of the cache is determined by the parameters provided by the
job and it is located in the local host. When possible Aeon uses these cache
for data caching.

For supporting data streaming in the above cases, a fixed
[hostPath][hostpath-pv-type] or [local][local-pv-type]
volume source type backed PV can act as a cache for the data.
In this case, controller will be responsible to make the PVC ready as soon as a
`hostPath` or `local` backed PV is created.

__Job output:__ When required, the job output can be gathered in a volume as
long as the backing file system supports `ReadWriteOnce` or `ReadWriteMany` access
mode.

__Garbage collection:__ The controller is responsible to evict unused data
from the node based on metrics such as disk pressure. Similarly, the controller
will also delete unused PVs and PVCs. When deleting PVs and PVCs, we will take
the reclaiming policies for a PV into consideration. A simple mechanism such as least
recently used (LRU) will be used to determine the order in which the data set,
PV and PVCs should be evicted.

## Relationship Between Volume and Data

The relationship between a volume and data is established using
`volumeConfig.sourceType` and a new data handler for that source type.

As the name implies, `volumeConfig.sourceType` provides the type of the data source
(e.g., S3 or NFS). The data handler for each source type provides the call-back
functions for a `volumeConfig` of that particular source type. These call-back
functions provide the logic to be executed when the CR containing the
`volumeConfig` is added, updated or deleted. This data handler should implement
the `DataHandler` interface in [handlers.go][handler-interface].

For each `sourceType`, a new data handler must be implemented. For more
information on adding a new data handler, read the [developer manual][dev-doc].

## Source Type Support Status

Brief description of source type support is provided below. For more
information on usage, refer to the [user manual][user-doc].

| Source Type            | Phase    | Description                                                                                                   |
| :----------------------| :--------| :-------------------------------------------------------------------------------------------------------------|
| S3-Dev                 | Supported| VCK will download the files from a specified S3 bucket and make it available for consumption in a node. This source type should only be used for development and testing purposes.      |
| S3                     | Supported| VCK will download the files from a specified S3 bucket and provide nodes where hostPath volumes can be used.  |
| NFS                    | Supported| VCK will make the specified path from an NFS server available for consumption.                                |
[ [Pachyderm][pachyderm] | Supported| VCK will download the pachyderm repo data and make it available for consumption on a specified number of nodes|
| [Aeon][aeon]           | Design   | -                                                                                                             |

[pv]: https://kubernetes.io/docs/concepts/storage/persistent-volumes/
[pvc]: https://kubernetes.io/docs/concepts/storage/persistent-volumes/#persistentvolumeclaims
[crd]: https://kubernetes.io/docs/tasks/access-kubernetes-api/extend-api-custom-resource-definitions/
[cr]: https://kubernetes.io/docs/concepts/api-extension/custom-resources/
[local-pv-type]: https://kubernetes.io/docs/concepts/storage/volumes/#local
[hostpath-pv-type]: https://kubernetes.io/docs/concepts/storage/volumes/#hostpath
[vol-sched]: https://github.com/kubernetes/features/issues/490
[aeon]: https://github.com/NervanaSystems/aeon
[handler-interface]: ../pkg/handlers/handlers.go
[dev-doc]: dev.md
[node-aff]: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/#node-affinity-beta-feature
[hostPath]: https://kubernetes.io/docs/concepts/storage/volumes/#hostpath
[vols]: https://kubernetes.io/docs/concepts/storage/volumes/
[volsources]: https://github.com/kubernetes/api/blob/master/core/v1/types.go#L250
[tolerations]: https://kubernetes.io/docs/concepts/configuration/taint-and-toleration/#concepts
[pachyderm]: http://pachyderm.io
