# Cleaning Up

The following commands will delete the Kubernetes objects associated with this initializer.

```
kubectl delete initializerconfiguration vck
```

```
kubectl delete deployment vck-initializer helloworld helloworld-with-annotation
```