# VCK Initializer

The VCK Initializer is a [Kubernetes initializer](https://kubernetes.io/docs/admin/extensible-admission-controllers/#what-are-initializers) that injects the vck volume mounts into the deployment

## Usage

```
vck-initializer -h
```
```
Usage of vck-initializer:
  -annotation string
    	The annotation to trigger initialization (default "initializer.kubernetes.io/vck")
  -initializer-name string
    	The initializer name (default "vck.initializer.kubernetes.io")
  -namespace string
    	The configuration namespace (default "vck")
  -require-annotation
    	Require annotation for initialization
```
