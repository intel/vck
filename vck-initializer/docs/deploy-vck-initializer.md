# Deploy The VCK Initializer

The VCK Initializer is a [Kubernetes Initializer](https://kubernetes.io/docs/admin/extensible-admission-controllers/#what-are-initializers) that injects an [Envoy](https://lyft.github.io/envoy) proxy into Deployments based on containers and volumes defined in a [ConfigMap](https://kubernetes.io/docs/tasks/configure-pod-container/configure-pod-configmap).

## Install

### Create the VCK Initializer Deployment

Deploy the `VCK-initializer` controller:

```
kubectl apply -f deployments/vck-initializer.yaml
```

The `vck-initializer` Deployment sets pending initializers to an empty list which bypasses initialization. This prevents the VCK Initializer from getting stuck waiting for initialization, which can happen if the `vck` [Initialization Configuration](initializing-deployments.md#create-the-vck-initializer-InitializerConfiguration) is created before the `vck-initializer` Deployment.

```
apiVersion: apps/v1beta1
kind: Deployment
metadata:
  initializers:
    pending: []
```

At this point the VCK Initializer is ready to initialize new Deployments.
