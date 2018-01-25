package controller

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"

	"github.com/NervanaSystems/kube-controllers-go/pkg/crd"
)

// Hooks is the callback interface that defines controller behavior.
type Hooks interface {
	Add(obj interface{})
	Update(oldObj, newObj interface{})
	Delete(obj interface{})
}

// source defines the interface for getting the source
type source interface {
	GetSource(controller *Controller, namespace string) *cache.ListWatch
}

// sourceImpl is the struct which should be used to get the source for watching and listing CRs
type sourceImpl struct{}

func (s *sourceImpl) GetSource(controller *Controller, namespace string) *cache.ListWatch {
	return cache.NewListWatchFromClient(
		controller.Client,
		controller.CRD.Definition.Spec.Names.Plural,
		namespace,
		fields.Everything())
}

// handlerFuncs returns an instance of the handler functions type
// needed to create an informer based on the supplied controller hooks.
func handlerFuncs(h Hooks) cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc:    h.Add,
		UpdateFunc: h.Update,
		DeleteFunc: h.Delete,
	}
}

// Controller watches a resource and delegates create/update/delete events
// to a set of supplied callback functions.
type Controller struct {
	CRD    *crd.Handle
	Hooks  Hooks
	Client rest.Interface
	Scheme *runtime.Scheme
	source source
}

// New returns a new Controller.
func New(handle *crd.Handle, hooks Hooks, client rest.Interface) *Controller {
	return &Controller{
		CRD:    handle,
		Hooks:  hooks,
		Client: client,
		source: &sourceImpl{},
	}
}

// Run starts a resource controller
func (c *Controller) Run(ctx context.Context, namespace string) error {
	fmt.Print("Watch objects\n")

	// Create source
	source := c.source.GetSource(c, namespace)

	// Watch objects
	_, err := c.watch(ctx, namespace, source)
	if err != nil {
		fmt.Printf("Failed to register watch for resource: %v\n", err)
		return err
	}

	<-ctx.Done()
	return ctx.Err()
}

func (c *Controller) watch(ctx context.Context, namespace string, source *cache.ListWatch) (cache.Controller, error) {

	_, cacheController := cache.NewInformer(
		source,

		// The object type.
		c.CRD.ResourceType,

		// resyncPeriod
		// Every resyncPeriod, all resources in the cache will retrigger events.
		// Set to 0 to disable the resync.
		0,

		// Your custom resource event handlers.
		handlerFuncs(c.Hooks),
	)

	go cacheController.Run(ctx.Done())
	return cacheController, nil
}
