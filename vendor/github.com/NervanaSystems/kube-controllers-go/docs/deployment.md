Running a controller
=======================

## Requirements for an runnable controller

The preferred method for installation of CRD controllers is with [Nervana Helm](https://github.com/NervanaSystems/nervana-helm). What follows are the requirements for deploying a controller along with a Nervana Cloud environment.

1. Make the version you'd like to deploy available in Nervana's private GCR repository. This is done automatically for any CircleCI build. You can find the tag in the output of the `Run make` section of the build's CirceCI page, or by running `git describe` from the target branch:
   ```shell
   kube-controllers-go $ git describe
   v0.1.0-8-g82173ec
   ```
1. In Nervana Helm:
   1. Create a new folder whose name matches the controller's name, at the root of the repository.
      ```
      nervana-helm $ mkdir {controller-name}
       ```
   1. Create the chart and template files used for Helm.  An example for the contents of these files can be found [here](https://github.com/NervanaSystems/nervana-helm/tree/sp-crd-controller/stream-prediction-controller).
      ```shell
      nervana-helm/{controller-name} $ tree
      {controller-name}
      ├── Chart.yaml
      ├── templates
      │   └── {controller-name}-dp.yaml
      └── values.yaml -> ../values.yaml
      ```
   1. Add the new controller to the appropriate [environment files](https://github.com/NervanaSystems/nervana-helm/tree/sp-crd-controller/environments). These files list which components to deploy for a given environment. If the new controller is not present in an environment's list, it will not be deployed with the installation scripts.  At the time of this writing, CRD controllers are only a member of the `k8s-experimental` environment:
      ```shell
      nervana-helm $ cat environments/k8s-experimental-group1.charts
      mysql
      redis
      mongo
      helium-proxy
      secrets
      krypton-temp
      nds-db-init
      helium-db-init
      nervana-ingress
      helium
      helium-web
      launchpad
      ncs
      nds
      helium-celery
      nds-celery
      stream-prediction-controller
      nervana-helm $ cat environments/k8s-experimental-values.yaml
      nervana_namespace: ${NERVANA_NAMESPACE}
      ```
   1. Add a stanza for the new controller in the top-level `values.yaml` file, providing the necessary configuration information.  Here's an example from the stream prediction controller:
      ```yaml
      stream_prediction_controller:
        registry: *registry
        org: *org
        repo: "stream-prediction-controller"
        tag: "v0.1.0-5-g62e2acc"
        loglevel: "4"
      ```
      **NOTE**: The tag comes from the image's tag, discoverable, as mentioned above by locating it in the build output, or using `git describe`.

## Running the controller

1. [Install Nervana Helm's dependencies](https://github.com/NervanaSystems/nervana-helm/blob/master/docs/install.md).
   Don't forget to ask for the 🔑 from a DevOps representative.  Without the keys, builds will proceed, but their failure modes can be specious.
1. Once these requirements are met, a controller can be run using the instructions in Nervana Helm's documentation, modulo one minor change.  At the time of this writing, controller deployment is done through a custom environment, namely `k8s-experimental`.  When using the `install_nervana.sh` script, one also needs to set the `NERVANA_ENVIRONMENT` environment variable to the correct environment:
   ```shell
   nervana-helm $ NERVANA_ENVIRONMENT=k8s-experimental ./scripts/install_nervana.sh all --wait
   ```

## Rollouts and Upgrades

Given the preceding prerequisites, rolling out upgrades for a controller is as simple as updating the tag for the controller in the top-level `values.yaml` file, and rerunning the `install_nervana.sh` script, pointed at specifically the updated controller:
```shell
nervana-helm $ cat values.yaml
# ...
stream_prediction_controller:
  registry: *registry
  org: *org
  repo: "stream-prediction-controller"
  tag: "v1.0.0-stable" # an updated tag
  loglevel: "4"
nervana-helm $ NERVANA_ENVIRONMENT=k8s-experimental ./scripts/install_nervana.sh {updated-controller} --wait
```
