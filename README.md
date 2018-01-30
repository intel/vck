# Nervana Volume Controller
# Docker Make

If you don't want to install the golang toolchain locally and you have docker installed, you can run `docker_make` instead of `make`:

```
$ ./docker_make dep-ensure
dep ensure

$ ./docker_make code-generation
/go/bin/deepcopy-gen --output-base=/go/src --input-dirs=github.com/NervanaSystems/kube-volume-controller/pkg/apis/cr/v1/...

$ ./docker_make lint
gometalinter --config=./lint.json --vendor .
# Disabling golint for apis since it conflicts with the deepcopy-gen
# annotations.
gometalinter --config=./lint.json --disable=golint ./pkg/apis/...
gometalinter --config=./lint.json ./pkg/hooks/...


$ ./docker_make test
go test --cover ./...
?   	github.com/NervanaSystems/kube-volume-controller	[no test files]
ok  	github.com/NervanaSystems/kube-volume-controller/pkg/apis/cr/v1	0.166s	coverage: 53.4% of statements
?   	github.com/NervanaSystems/kube-volume-controller/pkg/hooks	[no test files]
# go test --cover .
# go test --cover ./pkg/apis/...
# go test --cover ./pkg/hooks/...

$ ./docker_make build
dep ensure
/go/bin/deepcopy-gen --output-base=/go/src --input-dirs=github.com/NervanaSystems/kube-volume-controller/pkg/apis/cr/v1/...
go build -gcflags "-N -l" github.com/NervanaSystems/kube-volume-controller
```
