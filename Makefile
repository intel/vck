.PHONY: docker test

all: docker

VERSION := $(shell git describe --tags --always --dirty)

GOOGLE_PROJECT_ID=
GOOGLE_AUTH=
IMAGE_NAME=kube-volume-controller

docker:
	docker build \
		-t $(IMAGE_NAME):$(VERSION) \
		-t $(IMAGE_NAME):latest \
		.

lint:
	gometalinter --config=./lint.json --vendor .
	# Disabling golint for apis since it conflicts with the deepcopy-gen
	# annotations.
	gometalinter --config=./lint.json --disable=golint ./pkg/apis/...
	gometalinter --config=./lint.json ./pkg/hooks/...

test:
	go test --cover ./...
	# go test --cover .
	# go test --cover ./pkg/apis/...
	# go test --cover ./pkg/hooks/...

code-generation:
	/go/bin/deepcopy-gen --output-base=/go/src --input-dirs=github.com/NervanaSystems/kube-volume-controller/pkg/apis/cr/v1/...

push-image-preflight:
ifeq ($(GOOGLE_PROJECT_ID),)
	$(error GOOGLE_PROJECT_ID must be set)
endif
ifeq ($(GOOGLE_AUTH),)
	$(error GOOGLE_AUTH must be set)
endif
	echo "$(GOOGLE_AUTH)" | base64 --decode > /tmp/gcp-key.json
	gcloud auth activate-service-account --key-file /tmp/gcp-key.json
	gcloud config set project "$(GOOGLE_PROJECT_ID)"

push-image: push-image-preflight docker
	@ echo "tagging container"
	docker tag $(IMAGE_NAME):$(VERSION) gcr.io/$(GOOGLE_PROJECT_ID)/$(IMAGE_NAME):$(VERSION)
	@ echo "pushing container to gcr.io"
	gcloud docker -- push gcr.io/$(GOOGLE_PROJECT_ID)/$(IMAGE_NAME):$(VERSION)
