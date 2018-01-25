package main

import (
	"fmt"

	crv1 "github.com/NervanaSystems/kube-controllers-go/cmd/example-controller/apis/cr/v1"
	"github.com/NervanaSystems/kube-controllers-go/pkg/crd"
	"github.com/NervanaSystems/kube-controllers-go/pkg/states"
)

// exampleHooks implements controller.Hooks.
type exampleHooks struct {
	crdClient crd.Client
}

func (c *exampleHooks) Add(obj interface{}) {
	example := obj.(*crv1.Example)
	fmt.Printf("[CONTROLLER] OnAdd %s\n", example.ObjectMeta.SelfLink)
	// NEVER modify objects from the store. It's a read-only, local cache.
	// You can use DeepCopy() to make a deep copy of original object and modify
	// this copy or create a copy manually for better performance.
	exampleCopy := example.DeepCopy()
	exampleCopy.Status = crv1.ExampleStatus{
		State:   states.Completed,
		Message: "Successfully processed by controller",
	}

	_, err := c.crdClient.Update(exampleCopy)
	if err != nil {
		fmt.Printf("ERROR updating status: %v\n", err)
	} else {
		fmt.Printf("UPDATED status: %#v\n", exampleCopy)
	}
}

func (c *exampleHooks) Update(oldObj, newObj interface{}) {
	oldExample := oldObj.(*crv1.Example)
	newExample := newObj.(*crv1.Example)
	fmt.Printf("[CONTROLLER] OnUpdate oldObj: %s\n", oldExample.ObjectMeta.SelfLink)
	fmt.Printf("[CONTROLLER] OnUpdate newObj: %s\n", newExample.ObjectMeta.SelfLink)
}

func (c *exampleHooks) Delete(obj interface{}) {
	example := obj.(*crv1.Example)
	fmt.Printf("[CONTROLLER] OnDelete %s\n", example.ObjectMeta.SelfLink)
}
