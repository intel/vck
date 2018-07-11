# Operator Manual: Volume Controller for Kubernetes (VCK)
  
  * [Operator Manual: Volume Controller for Kubernetes (VCK)](#operator-manual-volume-controller-for-kubernetes-vck)
    * [Prerequisites](#prerequisites)
    * [Before You Begin](#before-you-begin)
    * [Installing the Controller for the first time](#installing-the-controller-for-the-first-time)
      * [Installing VCK Controller in your namespaces](#installing-vck-controller-in-your-namespaces)
      * [Custom Helm Options in VCK](#custom-helm-options-in-vck)
      * [Installing VCK from tip of a branch](#installing-vck-from-tip-of-a-branch)
      * [Deleting VCK Controller from your namespace](#deleting-vck-controller-from-your-namespace)

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

## Installing the Controller for the first time

Clone the repo and specify the namespace within `<>` to install VCK:

```sh
$ git clone git@github.com:IntelAI/vck.git
$ cd vck
$ helm install helm-charts/kube-volume-controller/ -n vck --wait \
  --set namespace=<vck_namespace>

NAME:   vck
LAST DEPLOYED: Tue Feb  6 12:58:50 2018
NAMESPACE: vck-testing
STATUS: DEPLOYED

RESOURCES:
==> v1/StorageClass
NAME       PROVISIONER                   AGE
vck-local  kubernetes.io/no-provisioner  11s

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

* Installed in vck
* Cluster Role: true
* Storage Class: true
```

The above command will install the VCK controller as a deployment, the storage
class for dynamic provisioning of persistent volumes and persistent volume
claims, a service account for the controller and all the RBAC related objects
such as cluster role and cluster role binding.

If the installation was successful, VCK is ready to use. The installation can be
verified using the command shown below.

```sh
$ kubectl get crd
NAME                            AGE
volumemanagers.vck.intelai.org   1h
```

### Installing VCK Controller in your namespaces

Once VCK is installed in one Kubernetes namespace and in order to use it in another namespace, VCK controller needs to be installed in the new namespace, but subsequent installations no longer require the `clusterrole`, or `storageclass` to be enabled:

```sh
$ YOUR_NAMESPACE=<your_namespace>
$ helm install helm-charts/kube-volume-controller/ -n vck-${YOUR_NAMESPACE} --wait \
  --set clusterrole.install=false \
  --set storageclass.install=false \
  --set crd.install=false \
  --set namespace=${YOUR_NAMESPACE}
```

### Custom Helm Options in VCK

Any helm chart values can be configured via `--set key=value` in the helm command. For example, you can specify VCK version via `--set tag="v0.1.0"` and enable logging via `--set log_level=4`:

```sh
$ helm install helm-charts/kube-volume-controller/ -n vck --wait \
  --set tag="v0.1.0" \
  --set log_level=4 \
  --set namespace=<vck_namespace>
```

#### Installing VCK from tip of a branch
You can also install `VCK` from feature branches if needed. In this case follow same steps as described above, but set the `tag` option this way:
```
--set-string tag="$(git describe --tags --always --dirty)"
```

### Deleting VCK Controller from your namespace
If you need to uninstall the Controller from your namespace try running the command:

```sh
$ YOUR_NAMESPACE=<your_namespace>
$helm delete --purge vck-${YOUR_NAMESPACE}
```

For a complete list of parameters, please review the [helm values file][helm-values] for additional information.

See [user manual][user-doc] for details on how to use VCK.

[helm-values]: ../helm-charts/kube-volume-controller/values.yaml
[user-doc]: user.md
[vol-sched]: https://github.com/kubernetes/features/issues/490
[helm]: https://docs.helm.sh/using_helm/
[kubectl]: https://kubernetes.io/docs/tasks/tools/install-kubectl/
