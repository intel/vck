FROM kube-controllers-go-dep

ADD . /go/src/github.com/NervanaSystems/kube-controllers-go
WORKDIR /go/src/github.com/NervanaSystems/kube-controllers-go
RUN go get github.com/kubernetes/gengo/examples/deepcopy-gen
RUN make code-generation
RUN make test
CMD /bin/bash
