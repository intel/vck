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
  github.com/nervanasystems/kubevolumecontroller/pkg/client github.com/NervanaSystems/kube-volume-controller/pkg/apis \
  cr:v1 --go-header-file pkg/apis/cr/v1/doc.go.txt

# This whole hack is because the autogens don't like upper case letters as well as hyphens. This is sad.
# This issue: https://github.com/kubernetes/code-generator/issues/22 needs to be resolved in to remove this hack.
mv /go/src/github.com/nervanasystems/kubevolumecontroller/pkg/client pkg/
find pkg/client -name "*.go" | xargs -n1 sed -i 's\nervanasystems/kubevolumecontroller\NervanaSystems/kube-volume-controller\g'
find pkg/client -name "*.go" | xargs -n1 sed -i 's\nervanasystems/kube-volume-controller\NervanaSystems/kube-volume-controller\g'
sed -i 's\volumeManager.Name)\volumeManager.Name())\g' pkg/client/clientset/versioned/typed/cr/v1/volumemanager.go
# To use your own boilerplate text append:
#   --go-header-file ${SCRIPT_ROOT}/hack/custom-boilerplate.go.txt
