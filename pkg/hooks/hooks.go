package hooks

import (
	"github.com/golang/glog"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/NervanaSystems/kube-controllers-go/pkg/crd"
	"github.com/NervanaSystems/kube-controllers-go/pkg/resource"
	crv1 "github.com/NervanaSystems/kube-volume-controller/pkg/apis/cr/v1"
)

// VolumeManagerHooks implements controller.Hooks interface
type VolumeManagerHooks struct {
	resourceClients []resource.Client
	crdClient       crd.Client
}

// NewVolumeManagerHooks creates and returns a new instance of the VolumeManagerHooks
func NewVolumeManagerHooks(crdClient crd.Client, resourceClients []resource.Client) *VolumeManagerHooks {
	return &VolumeManagerHooks{
		resourceClients: resourceClients,
		crdClient:       crdClient,
	}
}

// Add handles the addition of a new volume manager object
func (h *VolumeManagerHooks) Add(obj interface{}) {
	volumeManager, ok := obj.(*crv1.VolumeManager)
	if !ok {
		glog.Errorf("object received is not of type VolumeManager %v", obj)
		return
	}
	glog.V(4).Infof("Volume Manager add hook - got: %v", volumeManager)
}

// Update handles the update of a volume manager object
func (h *VolumeManagerHooks) Update(oldObj, newObj interface{}) {
	newVolumeManager, ok := newObj.(*crv1.VolumeManager)
	if !ok {
		glog.Errorf("object received is not of type VolumeManager %v", newObj)
		return
	}

	oldVolumeManager, ok := oldObj.(*crv1.VolumeManager)
	if !ok {
		glog.Errorf("object received is not of type VolumeManager %v", oldObj)
		return
	}

	glog.V(4).Infof("Volume Manager update hook - got old: %v new: %v", oldVolumeManager, newVolumeManager)
}

// Delete handles the deletion of a volume manager object
func (h *VolumeManagerHooks) Delete(obj interface{}) {
	volumeManager, ok := obj.(*crv1.VolumeManager)
	if !ok {
		glog.Errorf("object received is not of type VolumeManager %v", obj)
		return
	}
	glog.V(4).Infof("Volume Manager add hook - got: %v", volumeManager)

	//Delete the resources using name for now.
	h.deleteResources(volumeManager)
}

func (h *VolumeManagerHooks) addResources(volumeManager *crv1.VolumeManager) error {
	// Add controller reference.
	// See https://kubernetes.io/docs/concepts/workloads/controllers/garbage-collection/
	// for more details on owner references.
	ownerRef := metav1.NewControllerRef(volumeManager, crv1.GVK)

	for _, resourceClient := range h.resourceClients {
		err := resourceClient.Create(volumeManager.Namespace(), struct {
			*crv1.VolumeManager
			metav1.OwnerReference
		}{
			volumeManager,
			*ownerRef,
		})
		if err != nil {
			glog.Errorf("received err: %v while creating object", err)
			return err
		}
	}
	glog.Infof("resource creation complete for volume manager \"%s\"", volumeManager.Name())
	return nil
}

func (h *VolumeManagerHooks) deleteResources(volumeManager *crv1.VolumeManager) {
	for _, resourceClient := range h.resourceClients {
		if err := resourceClient.Delete(volumeManager.Namespace(), volumeManager.Name()); err != nil {
			glog.Errorf("resource deletion failed for volume manager \"%s\": %v", volumeManager.Name(), err)
		}
	}
	glog.Info("resource deletion complete for volume manager \"%s\"", volumeManager.Name())
}
