# Kube-volume-controller(KVC) Ksonnet Registry

## Overview

This directory contains the KVC ksonnet [registry][2]. If you are unfamiliar with ksonnet, we recommend browsing [the official site][1] to gain more context.

##Install 

### Requirements

  * ksonnet version [0.9.2](https://ksonnet.io/#get-started) or later.
  * Kubernetes >= 1.8 [see here](https://github.com/kubeflow/tf-operator#requirements)

### Steps

In order to quickly set up all components, execute the following commands:

```commandline
# Create a namespace for kvc deployment
NAMESPACE=kvc
kubectl create namespace ${NAMESPACE}

# Initialize a ksonnet app. Set the namespace for it's default environment.
APP_NAME=kvc
ks init ${APP_NAME}
cd ${APP_NAME}
ks env set default --namespace ${NAMESPACE}

# Install KVC components
ks registry add kvc-ksonnet github.com/sudeshsh/experimental-kvc/tree/master/kvc-ksonnet

ks pkg install kvc-ksonnet/kube-volume-controller

# Create templates for kvc components
ks generate kvc kvc



# Deploy kvc
ks apply default -c kvc
```

## Usage

Please refer to the [KVC user guide](https://github.com/kubeflow/experimental-kvc/blob/master/docs/user.md)

[1]: https://ksonnet.io
[2]: https://ksonnet.io/docs/concepts#registry
[3]: https://ksonnet.io/#get-started
[4]: https://github.com/ksonnet/parts/blob/master/doc-gen/main.go
