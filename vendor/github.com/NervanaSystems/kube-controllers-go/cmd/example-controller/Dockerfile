FROM kube-controllers-go

WORKDIR /go/src/github.com/NervanaSystems/kube-controllers-go/cmd/example-controller
# -gcflags: disable compiler optimizations and inlining
RUN make code-generation
RUN go install -gcflags "-N -l" github.com/NervanaSystems/kube-controllers-go/cmd/example-controller
RUN make test
CMD /go/bin/example-controller
