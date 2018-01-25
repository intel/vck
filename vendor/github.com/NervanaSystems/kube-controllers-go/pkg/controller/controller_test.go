package controller

import (
	"context"
	"github.com/NervanaSystems/kube-controllers-go/pkg/crd"
	"github.com/stretchr/testify/assert"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	fcache "k8s.io/client-go/tools/cache/testing"
	"testing"
)

type FakeHook struct {
	addCalled    bool
	deleteCalled bool
	updateCalled bool
	counter      chan string
}

func (f *FakeHook) Add(obj interface{}) {
	f.counter <- "Add"
	f.addCalled = true
	return
}

func (f *FakeHook) Update(oldObj, newObj interface{}) {
	f.counter <- "Update"
	f.updateCalled = true
	return
}

func (f *FakeHook) Delete(obj interface{}) {
	f.counter <- "Delete"
	f.deleteCalled = true
	return
}

func TestController(t *testing.T) {

	counter := make(chan string, 3)
	hook := FakeHook{counter: counter}

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	// Create a fake source and a fake listWatcher to inject objects into.
	source := fcache.NewFakeControllerSource()
	fakeSource := &FakeSource{source}

	handle := &crd.Handle{
		ResourceType: &v1.Pod{},
	}
	controller := New(handle, &hook, nil)
	controller.source = fakeSource

	go controller.Run(ctx, "")

	// Create a fake pod
	pod := &v1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "pod1"}}
	source.Add(pod)

	// Modify the pod
	pod.ObjectMeta.Name = "pod2"
	source.Modify(pod)

	// Wait till they are processed
	for i := 0; i < 2; i++ {
		<-counter
	}

	// Get one of the objects to delete
	listObj, err := source.List(metav1.ListOptions{})
	assert.Nil(t, err)
	assert.NotNil(t, listObj)

	list := listObj.(*v1.List)
	assert.NotNil(t, list)
	assert.Equal(t, 2, len(list.Items))
	objToDelete := list.Items[0]
	assert.NotNil(t, objToDelete)

	// Delete the object and wait for it to be deleted
	source.Delete(objToDelete.Object)
	<-counter

	// Assert all of them were called.
	assert.True(t, hook.addCalled)
	assert.True(t, hook.updateCalled)
	assert.True(t, hook.deleteCalled)
}

type FakeSource struct {
	source *fcache.FakeControllerSource
}

func (f *FakeSource) GetSource(controller *Controller, namespace string) *cache.ListWatch {
	listWatch := &cache.ListWatch{
		ListFunc:  f.source.List,
		WatchFunc: f.source.Watch,
	}
	return listWatch
}
