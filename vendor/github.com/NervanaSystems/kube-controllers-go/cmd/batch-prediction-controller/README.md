# batch-prediction-controller

The batch-prediction-controller contains all the logic to handle batch prediction
jobs in Kubernetes, including execution and synchronization. It implements 
call backs for addition, update and deletion of batch prediction jobs and takes 
appropriate actions based on the states in the `Spec` and `Status`. 

## Build

- Requires `docker`.

```
$ make
```

## Run

```
$ docker run batch-prediction-controller:v0.1.0
```
