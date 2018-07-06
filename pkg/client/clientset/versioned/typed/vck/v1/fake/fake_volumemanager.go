/*
<insert-license-here>
*/package fake

import (
	vck_v1 "github.com/ppkube/vck/pkg/apis/vck/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeVolumeManagers implements VolumeManagerInterface
type FakeVolumeManagers struct {
	Fake *FakeVckV1
	ns   string
}

var volumemanagersResource = schema.GroupVersionResource{Group: "vck.intelai.org", Version: "v1", Resource: "volumemanagers"}

var volumemanagersKind = schema.GroupVersionKind{Group: "vck.intelai.org", Version: "v1", Kind: "VolumeManager"}

// Get takes name of the volumeManager, and returns the corresponding volumeManager object, and an error if there is any.
func (c *FakeVolumeManagers) Get(name string, options v1.GetOptions) (result *vck_v1.VolumeManager, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(volumemanagersResource, c.ns, name), &vck_v1.VolumeManager{})

	if obj == nil {
		return nil, err
	}
	return obj.(*vck_v1.VolumeManager), err
}

// List takes label and field selectors, and returns the list of VolumeManagers that match those selectors.
func (c *FakeVolumeManagers) List(opts v1.ListOptions) (result *vck_v1.VolumeManagerList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(volumemanagersResource, volumemanagersKind, c.ns, opts), &vck_v1.VolumeManagerList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &vck_v1.VolumeManagerList{}
	for _, item := range obj.(*vck_v1.VolumeManagerList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested volumeManagers.
func (c *FakeVolumeManagers) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(volumemanagersResource, c.ns, opts))

}

// Create takes the representation of a volumeManager and creates it.  Returns the server's representation of the volumeManager, and an error, if there is any.
func (c *FakeVolumeManagers) Create(volumeManager *vck_v1.VolumeManager) (result *vck_v1.VolumeManager, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(volumemanagersResource, c.ns, volumeManager), &vck_v1.VolumeManager{})

	if obj == nil {
		return nil, err
	}
	return obj.(*vck_v1.VolumeManager), err
}

// Update takes the representation of a volumeManager and updates it. Returns the server's representation of the volumeManager, and an error, if there is any.
func (c *FakeVolumeManagers) Update(volumeManager *vck_v1.VolumeManager) (result *vck_v1.VolumeManager, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(volumemanagersResource, c.ns, volumeManager), &vck_v1.VolumeManager{})

	if obj == nil {
		return nil, err
	}
	return obj.(*vck_v1.VolumeManager), err
}

// Delete takes name of the volumeManager and deletes it. Returns an error if one occurs.
func (c *FakeVolumeManagers) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(volumemanagersResource, c.ns, name), &vck_v1.VolumeManager{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeVolumeManagers) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(volumemanagersResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &vck_v1.VolumeManagerList{})
	return err
}

// Patch applies the patch and returns the patched volumeManager.
func (c *FakeVolumeManagers) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *vck_v1.VolumeManager, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(volumemanagersResource, c.ns, name, data, subresources...), &vck_v1.VolumeManager{})

	if obj == nil {
		return nil, err
	}
	return obj.(*vck_v1.VolumeManager), err
}
