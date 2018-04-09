// @apiVersion 0.1
// @name io.ksonnet.pkg.kvc
// @description kvc main components
// @shortDescription kvc main components.
// @param name string Name to give to each of the components
// @optionalParam namespace string null Namespace to use for the components. It is automatically inherited from the environment if not set.
// @optionalParam cluster_role string true Install cluster role if necessary. Note: ClusterRole is a cluster-scoped object. It might already be installed.
// @optionalParam storage_class string true Install storage class when necessary. Note: StorageClass is a cluster-scoped object. It might already be installed.
// @optionalParam CRD string true Install CRD when necessary. Note: CRD is a cluster-scoped object. It might already be installed.
// @optionalParam log_level number 0 Enable for verbose log.

local k = import "k.libsonnet";
local kvc = import "kubeflow/tf-job/kvc.libsonnet";

local registry = "";
local org = "volumecontroller";
local repo = "kube-volume-controller";
local tag= "v0.1.0";

local updatedParams = params {
  namespace: if params.namespace == "null" then env.namespace else params.namespace,
};

local clusterRole = import "param://cluster_role";
local storageClass = import "param://storage_class";
local crd = import "param://CRD";
local logLevel = import "param://log_level";
local image = registry+org+"/"+repo+":"+tag;
local namespace = updatedParams.namespace;


// updatedParams uses the environment namespace if
// the namespace parameter is not explicitly set



local flags = [ "/kvc",
                "--podFile=/kvc-templates/pod.tmpl",
                "--pvFile=/kvc-templates/pv.tmpl", 
                "--pvcFile=/kvc-templates/pvc.tmpl",
                if logLevel != 0 then
                "--v="+logLevel,
                namespace];

std.prune(k.core.v1.list.new([
  if storageClass == "true" then
  kvc.parts.kvcStroageClass,
  if clusterRole == "true" then
  kvc.parts.kvcClusterRole,
  if clusterRole == "true" then
  kvc.parts.kvcClusterRoleBinding(namespace),
  if crd == "true" then
  kvc.parts.kvcCRD,
  kvc.parts.kvcServiceAccount(namespace),
  kvc.parts.kvcDeployment(namespace,image,flags),
]))

