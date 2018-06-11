package controller

import (
	"context"
	"fmt"

	vckv1_client "github.com/IntelAI/vck/pkg/client/clientset/versioned"
	vckv1_informer "github.com/IntelAI/vck/pkg/client/informers/externalversions"
	"k8s.io/client-go/tools/cache"
)

// Hooks is the callback interface that defines controller behavior.
type Hooks interface {
	Add(obj interface{})
	Update(oldObj, newObj interface{})
	Delete(obj interface{})
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
	Hooks  Hooks
	Client vckv1_client.Interface
}

// New returns a new Controller.
func New(hooks Hooks, client vckv1_client.Interface) *Controller {
	return &Controller{
		Hooks:  hooks,
		Client: client,
	}
}

// Run starts a resource controller
func (c *Controller) Run(ctx context.Context, namespace string) error {
	/**
	TODO: We spawn a goroutine with each onAdd hook. Investigate if that can be avoided by using something like:
	https://github.com/kubernetes/sample-controller/blob/master/controller.go#L169-L173.
	*/
	fmt.Print("Started watching for VolumeManager CR objects.\n")

	// Watch objects
	c.watch(ctx, namespace)

	<-ctx.Done()
	return ctx.Err()
}

func (c *Controller) watch(ctx context.Context, namespace string) {

	informer := vckv1_informer.NewFilteredSharedInformerFactory(c.Client, 0, namespace, nil)
	informer.Vck().V1().VolumeManagers().Informer().AddEventHandler(handlerFuncs(c.Hooks))

	go informer.Start(ctx.Done())

}
