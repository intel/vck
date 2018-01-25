.PHONY: docker test

VERSION := $(shell git describe --tags --always --dirty)

GOOGLE_PROJECT_ID=
GOOGLE_AUTH=
IMAGE_NAME=kube-controllers-go
TARGET ?= test
GODEBUGGER ?= gdb

all: controllers

test: lint validate-schemas
	go test -cover -v ./pkg/...

dep:
	docker build \
		-t $(IMAGE_NAME)-dep:$(VERSION) \
		-t $(IMAGE_NAME)-dep:latest \
		-f Dockerfile.dep .

docker:
	docker build \
		-t $(IMAGE_NAME):$(VERSION) \
		-t $(IMAGE_NAME):latest .

controllers: stream-prediction model-training batch-prediction example

code-generation:
	/go/bin/deepcopy-gen --output-base=/go/src --input-dirs=github.com/NervanaSystems/kube-controllers-go/pkg/crd/fake/... --output-package=pkg/crd/fake
	/go/bin/deepcopy-gen --output-base=/go/src --input-dirs=github.com/NervanaSystems/kube-controllers-go/pkg/resource/fake/... --output-package=pkg/resource/fake

stream-prediction:
	(cd cmd/stream-prediction-controller && make)

model-training:
	(cd cmd/model-training-controller && make)

batch-prediction:
	(cd cmd/batch-prediction-controller && make)

example:
	(cd cmd/example-controller && make)

env-up: env-down
	docker-compose up -d
	docker-compose ps

env-down:
	docker-compose down
	# resources is mounted as ~/.kube in the test container. This removes the
	# artifacts created during testing.
	rm -rf resources/cache

dev:
	docker-compose exec --privileged $(TARGET) /bin/bash

debug:
	docker-compose exec --privileged $(TARGET) env GODEBUGGER=$(GODEBUGGER) /go/src/github.com/NervanaSystems/kube-controllers-go/scripts/godebug attach $(TARGET)

create-sp:
	docker-compose exec --privileged $(TARGET) /usr/local/bin/kubectl create -f /go/src/github.com/NervanaSystems/kube-controllers-go/api/crd/examples/stream-prediction-job-valid-1.json

delete-sp:
	docker-compose exec --privileged $(TARGET) /usr/local/bin/kubectl delete -f /go/src/github.com/NervanaSystems/kube-controllers-go/api/crd/examples/stream-prediction-job-valid-1.json

test-e2e: env-up
	docker-compose exec test ./resources/wait-port kubernetes 8080
	# Run the stream-prediction controller tests in a new container with
	# the same configuration as the service, inside the docker-compose
	# environment.
	docker-compose exec test kubectl delete streampredictions --all --namespace=e2e-test || true
	docker-compose run stream-prediction-controller make test-e2e
	# Run the model-training controller tests in a new container with
	# the same configuration as the service, inside the docker-compose
	# environment.
	docker-compose exec test kubectl delete modeltrainings --all --namespace=e2e-test
	docker-compose run model-training-controller make test-e2e

install-linter:
	go get github.com/alecthomas/gometalinter
	gometalinter --install

lint:
	gometalinter --config=lint.json --disable=golint ./pkg/...

validate-schemas:
	(cd api/crd && make)

push-images:
	@ (cd cmd/example-controller && \
		make push-image \
		  GOOGLE_AUTH=$(GOOGLE_AUTH) \
		  GOOGLE_PROJECT_ID=$(GOOGLE_PROJECT_ID))
	@ (cd cmd/stream-prediction-controller && \
		make push-image \
		  GOOGLE_AUTH=$(GOOGLE_AUTH) \
		  GOOGLE_PROJECT_ID=$(GOOGLE_PROJECT_ID))
	@ (cd cmd/model-training-controller && \
		make push-image \
		  GOOGLE_AUTH=$(GOOGLE_AUTH) \
		  GOOGLE_PROJECT_ID=$(GOOGLE_PROJECT_ID))
	@ (cd cmd/batch-prediction-controller && \
		make push-image \
		  GOOGLE_AUTH=$(GOOGLE_AUTH) \
		  GOOGLE_PROJECT_ID=$(GOOGLE_PROJECT_ID))
