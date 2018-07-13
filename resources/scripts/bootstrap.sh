#!/bin/sh

### PS: Assumes a namespace with name $1 exists

set -e

NAMESPACE=$1
NFS_NAME=nfs-${NAMESPACE}
PACHYDERM_NAME=pachyderm-${NAMESPACE}
MINIO_NAME=minio-${NAMESPACE}

MINIO_ACCESS_KEY_ID=$2
MINIO_SECRET_ACCESS_KEY=$3
INSTALL_HELM=$4

tryuntil() {
    COMMAND=$1
    TARGET=$2
    TRIES=${3:-100}
    until eval $1 || [ $TRIES -eq 0 ]; do
        sleep 10
        echo "Waiting for ${TARGET}. Tries: $TRIES/${3:-100}"
        TRIES=$(( TRIES-1 ))
    done

    if [ $TRIES -eq 0 ]; then
        exit 1
    fi
}

if [ "x$INSTALL_HELM" = "xtrue" ]; then
    # Install the service account and the cluster role binding
    echo "Initializing helm.."
    kubectl create serviceaccount tiller --namespace=kube-system
cat <<EOF | kubectl create -f -
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRoleBinding
metadata:
  name: tiller
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: cluster-admin
subjects:
  - kind: ServiceAccount
    name: tiller
    namespace: kube-system
EOF
    helm init --service-account tiller

    tryuntil "helm ls" helm
fi

# Install nfs, minio and pachyderm
helm install --wait --namespace ${NAMESPACE} --name ${NFS_NAME} stable/nfs-server-provisioner
helm install --wait --namespace ${NAMESPACE} --name ${MINIO_NAME} --set acccessKey=${MINIO_ACCESS_KEY_ID} --set secretKey=${MINIO_SECRET_ACCESS_KEY} stable/minio
helm install --wait --namespace ${NAMESPACE} --name ${PACHYDERM_NAME} stable/pachyderm

# Wait for all pods to be running.
tryuntil '! kubectl get pods -n '${NAMESPACE}' | grep -v NAME | grep -v Running' bootstrap 1000

# Bootstrap pachyderm and minio with data
helm install --name bootstrap /go/src/github.com/IntelAI/vck/helm-charts/bootstrap \
    --set namespace="${NAMESPACE}" \
    --set minio.bootstrap=true \
    --set minio.server_address="http://${MINIO_NAME}.${NAMESPACE}.svc:9000" \
    --set minio.access_key="${MINIO_ACCESS_KEY_ID}" \
    --set minio.secret_key="${MINIO_SECRET_ACCESS_KEY}" \
    --set pachyderm.bootstrap=true \
    --set pachyderm.address="pachd.${NAMESPACE}.svc:650"

# Wait for the pods to run to completion
tryuntil 'kubectl get pod minio-bootstrap -n '${NAMESPACE}' | grep Completed' "minio bootstrap"

kubectl delete pod minio-bootstrap -n "${NAMESPACE}" || true

tryuntil 'kubectl get pod pachyderm-bootstrap -n '${NAMESPACE}' | grep Completed' "pachyderm bootstrap"

kubectl delete pod pachyderm-bootstrap -n "${NAMESPACE}" || true