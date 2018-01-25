# model-training-controller

The model-training-controller contains all the logic to handle model training
jobs in Kubernetes, including execution and synchronization. It implements 
call backs for addition, update and deletion of model training jobs and takes 
appropriate actions based on the states in the `Spec` and `Status`. 

## Build

- Requires `docker`.

```
$ make
```

## Run

```
$ docker run model-training-controller:v0.1.0
```
