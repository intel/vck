/*
<insert-license-here>
*/package fake

import (
	v1 "github.com/ppkube/vck/pkg/client/clientset/versioned/typed/vck/v1"
	rest "k8s.io/client-go/rest"
	testing "k8s.io/client-go/testing"
)

type FakeVckV1 struct {
	*testing.Fake
}

func (c *FakeVckV1) VolumeManagers(namespace string) v1.VolumeManagerInterface {
	return &FakeVolumeManagers{c, namespace}
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *FakeVckV1) RESTClient() rest.Interface {
	var ret *rest.RESTClient
	return ret
}
