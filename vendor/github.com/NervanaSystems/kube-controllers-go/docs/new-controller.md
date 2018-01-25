# How to make a new controller

The first step is to validate your development environment by ensuring
you can run `make` successfully before making changes.

1. Choose a name for your controller. For now, we'll assume your
   controller is called "foo" and the CRD resource type it manages is
   called "Foo".

1. Create a copy of the example controller directory.
   ```
   cp -R cmd/example-controller cmd/foo-controller
   ```

1. Add a new build target for your controller to the top-level Makefile.
   ```
   foo:
   	(cd cmd/foo-controller && make)
   ```

1. Edit files in `cmd/foo-controller/apis/cr/v1`, replacing all
   instances of `type Example` and `type ExampleList` with
   `type Foo` and `type FooList`.

1. Edit `/cmd/foo-controller/Dockerfile` to update the source code path.

1. Edit `/cmd/foo-controller/Makefile` to update the Docker tag to
   `foo-controller:$(version)` and update the code generation path to `github.com/NervanaSystems/kube-controllers-go/cmd/foo-controller/apis/cr/v1`

1. Edit `/cmd/foo-controller/main.go`:
    1. Edit the `crv1` import to `crv1 "github.com/NervanaSystems/kube-controllers-go/cmd/foo-controller/apis/cr/v1"`
    1. Change all instances of `crv1.Example` and `crv1.ExampleList` to
       `crv1.Foo` and `crv1.FooList`.
    1. Change the type of the hooks implementation from `exampleHooks`
       to `fooHooks`.

1. Edit `/cmd/foo-controller/hooks.go`:
    1. Edit the `crv1` import to `crv1 "github.com/NervanaSystems/kube-controllers-go/cmd/foo-controller/apis/cr/v1"`
    1. Change `type exampleHooks struct {` to `type fooHooks struct {`.
    1. Change all instances of `crv1.Example` and `crv1.ExampleList` to
       `crv1.Foo` and `crv1.FooList`.

1. Try to build your new controller!
   ```
   make foo
   ```

1. **[OPTIONAL]:** Add your controller to the `docker-compose.yml` file to
   deploy it locally as part of `make env-up`.
