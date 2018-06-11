local k = import "k.libsonnet";

{
  parts:: {
    stroageClass::
    {
      apiVersion: "storage.k8s.io/v1",
      kind: "StorageClass",
      metadata: {
        name: "kvc"
      },
      provisioner: "kubernetes.io/no-provisioner",
      volumeBindingMode: "WaitForFirstConsumer"
    },
    serviceAccount(namespace)::
    {
      apiVersion: "v1",
      kind: "ServiceAccount",
      metadata: {
        name: "kvc",
        namespace: namespace,
        labels: {
          app: "kvc"
        }
      }
    },
    clusterRole::{
      apiVersion: "rbac.authorization.k8s.io/v1",
      kind: "ClusterRole",
      metadata: {
        name: "kvc",
        labels: {
          app: "kvc"
        }
      },
      rules: [
        {
          apiGroups: ["kvc.kubeflow.org"],
          resources: ["volumemanagers"],
          verbs: ["*"]
        },
        {
          apiGroups: ["apiextensions.k8s.io"],
          resources: ["customresourcedefinitions"],
          verbs: ["*"]
        },
        {
          apiGroups: ["storage.k8s.io"],
          resources: ["storageclasses"],
          verbs: ["*"]
        },
        {
          apiGroups: [""],
          resources: ["configmaps", 
                      "pods", 
                      "services", 
                      "endpoints", 
                      "persistentvolumes", 
                      "persistentvolumeclaims", 
                      "events", 
                      "nodes"],
          verbs:[ "*"]
        },
        {
          apiGroups: ["apps",
                      "extensions"],
          resources: ["deployments"],
          verbs: ["*"]
        }
      ]
    },
    clusterRoleBinding(namespace)::
    {
      kind: "ClusterRoleBinding",
      apiVersion: "rbac.authorization.k8s.io/v1",
      metadata: {
        name: "kvc",
        labels: {
          app: "kvc"
          }
      },
      subjects: [
        {
          kind: "ServiceAccount",
          name: "kvc",
          namespace: namespace
        }
      ],
      roleRef: {
        apiGroup: "rbac.authorization.k8s.io",
        kind: "ClusterRole",
        name: "kvc"
      }
    },
    crd::{
      apiVersion: "apiextensions.k8s.io/v1beta1",
      kind: "CustomResourceDefinition",
      metadata: {
        name: "volumemanagers.kvc.kubeflow.org"
      },
      spec: {
        group: "kvc.kubeflow.org",
        names: {
          kind: "VolumeManager",
          listKind: "VolumeManagerList",
          plural: "volumemanagers",
          singular: "volumemanager"
        },
      scope: "Namespaced",
      version: "v1"
      }
    },
    deployment(namespace,image,args=[])::
    {
      apiVersion: "extensions/v1beta1",
      kind: "Deployment",
      metadata: {
        name: "kvc",
        namespace: namespace,
        labels: {
          app: "kvc",
          servicetype: "controller"
        }
      },
      spec: {
        replica: 1,
        strategy: {
          rollingUpdate: {
            maxSurge: 1,
            maxUnavailable: 0
          },
          type: "RollingUpdate"
        },
        template: {
          metadata: {
            labels: {
              app: "kvc",
              servicetype: "controller"
            }
          },
          spec: {
            serviceAccountName: "kvc",
            containers: [
              {
                name: "kvc",
                image: image,
                imagePullPolicy: "IfNotPresent",
                command: args,
              }
            ]
          }
        }
      }
    },
  },
}

