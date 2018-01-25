FROM golang:1.9.2

RUN mkdir -p /go/src/github.com/NervanaSystems
ADD . /go/src/github.com/NervanaSystems/kube-volume-controller

WORKDIR /go/src/github.com/NervanaSystems/kube-volume-controller

# Run code-gen for CRD.
RUN go get -u github.com/kubernetes/gengo/examples/deepcopy-gen
RUN make code-generation

# Install and run linter.
RUN go get -u github.com/alecthomas/gometalinter
RUN gometalinter --install
RUN make lint

# Run tests.
RUN make test

# Install the controller.
RUN go install -gcflags "-N -l" github.com/NervanaSystems/kube-volume-controller
CMD /go/bin/kube-volume-controller
