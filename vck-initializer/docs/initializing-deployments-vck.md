# Initializing Deployments Based On Metadata

It's possible to select which objects are initialized using metadata. The VCK Initializer is configured to only initialize Deployments with an `initializer.kubernetes.io/vck` annotation set to a non-empty value.


## Deploy the VCK Initializer

Deploy the VCK Initializer with the `-require-annotation` flag set. This will ensure the  volume manager data is only injected into Deployments with an `initializer.kubernetes.io/vck` annotation set to a non-empty value as given below.
```
"initializer.kubernetes.io/vck": '{
        "name": "vck-example3", \\created volumemanger name
        "id": "vol2",   \\id of the volumenamger to be injected to the dpeloyment
        "containers": ["helloworld-1"], \\ list of container to append the volumemount,can be set to empty to append to all containers
        "mount-path": "/var/datasets" \\ location of volumemount
      }'
```
```
kubectl apply -f deployments/vck-initializer.yaml
```

Create the `helloworld` Deployment:

```
kubectl apply -f deployments/helloworld.yaml 
```

Notice the `helloworld` Deployment has been initialized without injecting the VCK mounts:

```
kubectl describe deployment helloworld
```
```
Name:                   helloworld
Namespace:              vck
CreationTimestamp:      Thu, 05 Jul 2018 11:19:13 -0700
Labels:                 app=helloworld
Annotations:            deployment.kubernetes.io/revision=1
Selector:               app=helloworld
Replicas:               1 desired | 1 updated | 1 total | 1 available | 0 unavailable
StrategyType:           RollingUpdate
MinReadySeconds:        0
RollingUpdateStrategy:  25% max unavailable, 25% max surge
Pod Template:
  Labels:  app=helloworld
  Containers:
   helloworld:
    Image:      ubuntu:latest
    Port:       <none>
    Host Port:  <none>
    Command:
      /bin/bash
      -c
      --
    Args:
      while true; do sleep 30; done;
    Environment:  <none>
    Mounts:       <none>
  Volumes:        <none>
```

### Create the helloworld-with-annotation Deployment

```
kubectl apply -f deployments/helloworld-with-annotation.yaml
```

Notice the `helloworld-with-annotation` Deployment has been initialized with the VCK volumemounts:

```
kubectl describe deployment helloworld-with-annotation
```
```

Name:                   helloworld-with-annotation
Namespace:              vck
CreationTimestamp:      Thu, 05 Jul 2018 11:20:46 -0700
Labels:                 app=helloworld
                        envoy=true
Annotations:            deployment.kubernetes.io/revision=1
                        initializer.kubernetes.io/vck={ "name": "vck-example3", "id": "vol2", "containers": ["helloworld-1"], "mount-path": "/var/datasets" }
Selector:               app=helloworld,envoy=true
Replicas:               1 desired | 1 updated | 1 total | 1 available | 0 unavailable
StrategyType:           RollingUpdate
MinReadySeconds:        0
RollingUpdateStrategy:  25% max unavailable, 25% max surge
Pod Template:
  Labels:  app=helloworld
           envoy=true
  Containers:
   helloworld-1:
    Image:      ubuntu:latest
    Port:       <none>
    Host Port:  <none>
    Command:
      /bin/bash
      -c
      --
    Args:
      while true; do sleep 30; done;
    Environment:  <none>
    Mounts:
      /var/datasets from dataset-claim (rw)
   helloworld-2:
    Image:      ubuntu:latest
    Port:       <none>
    Host Port:  <none>
    Command:
      /bin/bash
      -c
      --
    Args:
      while true; do sleep 30; done;
    Environment:  <none>
    Mounts:       <none>
  Volumes:
   dataset-claim:
    Type:          HostPath (bare host directory volume)
    Path:          /var/datasets/vck-resource-bb444014-7e4c-11e8-9b69-0a580a50012e
    HostPathType:
```
