#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

rm -rf ./vendor/k8s.io/code-generator
go get -d github.com/kubernetes/code-generator/...
mv /go/src/k8s.io/code-generator ./vendor/k8s.io/
pushd vendor/k8s.io/code-generator
git checkout kubernetes-1.9.2
popd


SCRIPT_ROOT=$(dirname ${BASH_SOURCE})/..
CODEGEN_PKG=${CODEGEN_PKG:-$(cd ${SCRIPT_ROOT}; ls -d -1 ./vendor/k8s.io/code-generator 2>/dev/null || echo ../code-generator)}

# generate the code with:
# --output-base    because this script should also be able to run inside the vendor dir of
#                  k8s.io/kubernetes. The output-base is needed for the generators to output into the vendor dir
#                  instead of the $GOPATH directly. For normal projects this can be dropped.
${CODEGEN_PKG}/generate-groups.sh all \
  github.com/kubeflow/experimental-kvc/pkg/client github.com/kubeflow/experimental-kvc/pkg/apis \
  kvc:v1 --go-header-file pkg/apis/kvc/v1/doc.go.txt
