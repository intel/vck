FROM kube-controllers-go

WORKDIR /go/src/github.com/NervanaSystems/kube-controllers-go/cmd/stream-prediction-controller
ADD . .
# -gcflags: disable compiler optimizations and inlining
RUN make code-generation
RUN make test
RUN go install -gcflags "-N -l" github.com/NervanaSystems/kube-controllers-go/cmd/stream-prediction-controller
CMD /go/bin/stream-prediction-controller
