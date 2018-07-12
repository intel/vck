#!/bin/bash
#
# Copyright (c) 2018 Intel Corporation
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
# SPDX-License-Identifier: EPL-2.0
#


set -o errexit
set -o nounset
set -o pipefail

rm -rf ./vendor/k8s.io/code-generator
go get -d github.com/kubernetes/code-generator/...
mv $GOPATH/src/k8s.io/code-generator ./vendor/k8s.io/
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
  github.com/intelai/vck/pkg/client github.com/IntelAI/vck/pkg/apis \
  vck:v1alpha1 --go-header-file pkg/apis/vck/v1alpha1/doc.go.txt

# This hack is required as the autogens don't work for upper case letters in package names.
# This issue: https://github.com/kubernetes/code-generator/issues/22 needs to be resolved to remove this hack.
mv /go/src/github.com/intelai/vck/pkg/client pkg/
find pkg/client -name "*.go" | xargs -n1 sed -i 's\intelai/vck\IntelAI/vck\g'
