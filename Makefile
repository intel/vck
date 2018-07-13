#
# Copyright (c) 2018 Intel Corporation
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http:#www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
# SPDX-License-Identifier: EPL-2.0
#

.PHONY: docker test

all: test

VERSION := $(shell git describe --tags --always --dirty)

IMAGE_NAME=kube-volume-controller

docker:
	docker build \
		-t $(IMAGE_NAME):$(VERSION) \
		-t $(IMAGE_NAME):latest \
		.

prereq:
	go get -u \
		github.com/golang/dep/cmd/dep \
		github.com/alecthomas/gometalinter \
		github.com/kubernetes/gengo/examples/deepcopy-gen
	gometalinter --install

dep-ensure:
	dep ensure

build: prereq dep-ensure code-generation lint test
	go build -gcflags "-N -l" github.com/IntelAI/vck

lint:
	gometalinter --config=./lint.json --vendor .
	# Disabling golint for apis since it conflicts with the deepcopy-gen
	# annotations.
	gometalinter --config=./lint.json --disable=golint ./pkg/apis/...
	gometalinter --config=./lint.json ./pkg/hooks/...
	gometalinter --config=./lint.json ./pkg/controller/...
	gometalinter --config=./lint.json ./pkg/handlers/...
	gometalinter --config=./lint.json ./pkg/util/...
	gometalinter --config=./lint.json ./test/...

test:
	go test -v --cover ./pkg/resource/...
	go test -v --cover ./pkg/hooks/...
	go test -v --cover ./pkg/handlers/...
	go test -v --cover ./pkg/controller/...

test-e2e:
	go test -v ./test/e2e/...

code-generation:
	./hack/update-codegen.sh

push-image: docker
	@ echo "tagging container"
	docker tag $(IMAGE_NAME):$(VERSION) volumecontroller/$(IMAGE_NAME):$(VERSION)
	@ echo "pushing container to gcr.io"
	docker push volumecontroller/$(IMAGE_NAME):$(VERSION)
