# Best Practices: Kubernetes Volume Controller (KVC)

## Deploying KVC

We recommend a KVC deployment per namespace. The namespace for the deployment
of the controller and the namespace being watched by the controller should
be the same.

## Stock Datasets and Models

_Note: For the purposes of this document, a stock dataset or model are the most
frequently used datasets and models in the cluster._

We recommend that the stock datasets and models are pre-populated in each node
of the cluster if there are no restrictions on the disk space. Each node can 
be pre-populated with stocks datasets and models using the S3 source type and
setting the replicas equal to the number of nodes in the cluster.

After the pre-population, persistent volume claim (PVC) names and the corresponding
dataset or model it represents can be tabulated. The user has to only interact
with the PVC name depending upon the dataset or model of their choice. A PVC
can be used by any number of objects in Kubernetes.

## Security and Isolation

Recommendations around RBAC, service accounts and namespaces will be added soon.

