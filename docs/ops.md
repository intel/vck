# Operator Manual: Kubernetes Volume Controller (KVC)
  
  * [Operator Manual: Kubernetes Volume Controller (KVC)](#operator-manual-kubernetes-volume-controller-kvc)
    * [Prerequisites](#prerequisites)
    * [Before You Begin](#before-you-begin)
    * [Installing the Controller](#installing-the-controller)
      * [Installing KVC in multiple namespaces](#installing-kvc-in-multiple-namespaces)
      * [Custom Helm Options in KVC](#custom-helm-options-in-kvc)

## Prerequisites

- Kubernetes v1.9+ with [`VolumeScheduling`][vol-sched] feature gate enabled
- [Kubectl][kubectl]
- [Helm][helm]

## Before You Begin

Create your own namespace and set your current context to use that namespace.
The following commands can be used after editing the comments within `<>`.

```sh
$ kubectl create namespace <insert-namespace-here>

$ kubectl config set-context $(kubectl config current-context) --namespace=<insert-namespace-here>
```

## Installing the Controller

Clone the repo and specify the namespace within `<>` to install KVC:

```sh
$ git clone git@github.com:kubeflow/experimental-kvc.git
$ cd experimental-kvc
$ helm install helm-charts/kube-volume-controller/ -n kvc --wait \
  --set namespace=<kvc_namespace>

NAME:   kvc
LAST DEPLOYED: Tue Feb  6 12:58:50 2018
NAMESPACE: kvc-testing
STATUS: DEPLOYED

RESOURCES:
==> v1/StorageClass
NAME       PROVISIONER                   AGE
kvc-local  kubernetes.io/no-provisioner  11s

==> v1/ServiceAccount
NAME                    SECRETS  AGE
kube-volume-controller  1        11s

==> v1/ClusterRole
NAME                    AGE
kube-volume-controller  11s

==> v1/ClusterRoleBinding
NAME                    AGE
kube-volume-controller  11s

==> v1beta1/Deployment
NAME                    DESIRED  CURRENT  UP-TO-DATE  AVAILABLE  AGE
kube-volume-controller  1        1        1           1          11s

==> v1/Pod(related)
NAME                                    READY  STATUS   RESTARTS  AGE
kube-volume-controller-84bc5789c-9wtsr  1/1    Running  0         11s

Notes:
Kube-Volume-Controller v0.1.0

* Installed in kvc
* Cluster Role: true
* Storage Class: true
```

The above command will install the KVC controller as a deployment, the storage
class for dynamic provisioning of persistent volumes and persistent volume
claims, a service account for the controller and all the RBAC related objects
such as cluster role and cluster role binding.

If the installation was successful, KVC is ready to use. The installation can be
verified using the command shown below.

```sh
$ kubectl get crd
NAME                            AGE
volumemanagers.kvc.kubeflow.org   1h
```

### Installing KVC in multiple namespaces

KVC can be installed in multiple namespaces on a Kubernetes cluster. Once KVC is installed in one Kubernetes namespace, subsequent installations in additional namespaces no longer require the `clusterrole`, or `storageclass` to be enabled:

```
$ helm install helm-charts/kube-volume-controller/ -n kvc --wait \
  --set clusterrole.install=false \
  --set storageclass.install=false \
  --set crd.install=false \
  --set namespace=<kvc_namespace>
```

### Custom Helm Options in KVC

Any helm chart values can be configured via `--set key=value` in the helm command. For example, you can specify KVC version via `--set tag="v0.1.0"` and enable logging via `--set log_level=4`:

```
$ helm install helm-charts/kube-volume-controller/ -n kvc --wait \
  --set tag="v0.1.0" \
  --set log_level=4 \
  --set namespace=<kvc_namespace>
```

For a complete list of parameters, please review the [helm values file][helm-values] for additional information.

See [user manual][user-doc] for details on how to use KVC.

[helm-values]: ../helm-charts/kube-volume-controller/values.yaml
[user-doc]: user.md
[vol-sched]: https://github.com/kubernetes/features/issues/490
[helm]: https://docs.helm.sh/using_helm/
[kubectl]: https://kubernetes.io/docs/tasks/tools/install-kubectl/

