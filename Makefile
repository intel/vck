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

build: dep-ensure code-generation
	go build -gcflags "-N -l" github.com/NervanaSystems/kube-volume-controller

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
	echo "$(GOOGLE_AUTH)" | base64 --decode > /tmp/gcp-key.json
	gcloud auth activate-service-account --key-file /tmp/gcp-key.json
	gcloud config set project "$(GOOGLE_PROJECT_ID)"

push-image: push-image-preflight docker
	@ echo "tagging container"
	docker tag $(IMAGE_NAME):$(VERSION) gcr.io/$(GOOGLE_PROJECT_ID)/$(IMAGE_NAME):$(VERSION)
	@ echo "pushing container to gcr.io"
	gcloud docker -- push gcr.io/$(GOOGLE_PROJECT_ID)/$(IMAGE_NAME):$(VERSION)
