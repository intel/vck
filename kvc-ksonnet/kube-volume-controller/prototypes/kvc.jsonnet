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
// @optionalParam tag string cd7ea57 specify version for kvc

local k = import "k.libsonnet";
local kvc = import "kvc-ksonnet/kube-volume-controller/kvc.libsonnet";

local registry = "";
local org = "volumecontroller";
local repo = "kube-volume-controller";
local tag = import "param://tag";

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
  kvc.parts.stroageClass,
  if clusterRole == "true" then
  kvc.parts.clusterRole,
  if clusterRole == "true" then
  kvc.parts.clusterRoleBinding(namespace),
  if crd == "true" then
  kvc.parts.crd,
  kvc.parts.serviceAccount(namespace),
  kvc.parts.deployment(namespace,image,flags),
]))
