package hooks

import (
	"fmt"
	"github.com/golang/glog"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/NervanaSystems/kube-controllers-go/pkg/crd"
	"github.com/NervanaSystems/kube-controllers-go/pkg/states"
	crv1 "github.com/NervanaSystems/kube-volume-controller/pkg/apis/cr/v1"
	"github.com/NervanaSystems/kube-volume-controller/pkg/handlers"
)

// VolumeManagerHooks implements controller.Hooks interface
type VolumeManagerHooks struct {
	crdClient    crd.Client
	dataHandlers []handlers.DataHandler
}

// NewVolumeManagerHooks creates and returns a new instance of the VolumeManagerHooks
func NewVolumeManagerHooks(crdClient crd.Client, dataHandlers []handlers.DataHandler) *VolumeManagerHooks {
	return &VolumeManagerHooks{
		crdClient:    crdClient,
		dataHandlers: dataHandlers,
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

	volumeManagerCopy := volumeManager.DeepCopy()

	// If created with a Failed desired state. We immediately change the volume
	// manager status to Failed.
	if volumeManagerCopy.Spec.State == states.Failed {
		volumeManagerCopy.Status = crv1.VolumeManagerStatus{
			State:   volumeManagerCopy.Spec.State,
			Message: "Added with desired state as failed and controller marked volume manager as " + string(volumeManagerCopy.Spec.State),
		}

		h.crdClient.Update(volumeManagerCopy)
		return
	}

	// Mark the CR as pending before starting to invoke the handlers.
	volumeManagerCopy.Status = crv1.VolumeManagerStatus{
		State:   states.Pending,
		Message: fmt.Sprintf("Beginning sub-resource deployment"),
	}

	obj, err := h.crdClient.Update(volumeManagerCopy)
	if err != nil {
		glog.Warningf("error updating status for volume manager %s: %v\n", volumeManagerCopy.Name(), err)
		return
	}

	volumeManagerCopy, ok = obj.(*crv1.VolumeManager)
	if !ok {
		glog.Errorf("object received is not of type VolumeManager %v", obj)
		return
	}

	controllerRef := metav1.NewControllerRef(volumeManagerCopy, crv1.GVK)

	vClaims := []crv1.VolumeClaim{}
	for _, handler := range h.dataHandlers {
		for _, vConfig := range volumeManagerCopy.Spec.VolumeConfigs {
			if handler.GetSourceType() == vConfig.SourceType {
				vClaim := handler.OnAdd(volumeManagerCopy.Namespace(), vConfig, *controllerRef)
				vClaims = append(vClaims, vClaim)
			}
		}
	}

	for _, vClaim := range vClaims {
		// If any of the volume claim was not successful, mark the CR as Failed.
		if vClaim.Message != crv1.SuccessfulVolumeClaimMessage {
			volumeManagerCopy.Status = crv1.VolumeManagerStatus{
				VolumeClaims: vClaims,
				State:        states.Failed,
				Message:      fmt.Sprintf("failed to deploy all the sub-resources"),
			}

			_, err := h.crdClient.Update(volumeManagerCopy)
			if err != nil {
				glog.Warningf("error updating status for volume manager %s: %v\n", volumeManagerCopy.Name(), err)
				return
			}
		}
	}

	// Mark the CR as Running.
	volumeManagerCopy.Status = crv1.VolumeManagerStatus{
		VolumeClaims: vClaims,
		State:        states.Running,
		Message:      fmt.Sprintf("successfully deployed all sub-resources"),
	}

	_, err = h.crdClient.Update(volumeManagerCopy)
	if err != nil {
		glog.Warningf("error updating status for volume manager %s: %v\n", volumeManagerCopy.Name(), err)
		return
	}
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

	controllerRef := metav1.NewControllerRef(newVolumeManager, crv1.GVK)
	// Delete all the sub-resources if the CR is in a failed state.
	if newVolumeManager.Status.State == states.Failed {
		// Delete all the sub-resources.
		for _, handler := range h.dataHandlers {
			for _, vConfig := range newVolumeManager.Spec.VolumeConfigs {
				if handler.GetSourceType() == vConfig.SourceType {
					handler.OnDelete(newVolumeManager.Namespace(), vConfig, *controllerRef)
				}
			}
		}
	}

}

// Delete handles the deletion of a volume manager object
func (h *VolumeManagerHooks) Delete(obj interface{}) {
	volumeManager, ok := obj.(*crv1.VolumeManager)
	if !ok {
		glog.Errorf("object received is not of type VolumeManager %v", obj)
		return
	}
	glog.V(4).Infof("Volume Manager delete hook - got: %v", volumeManager)

	controllerRef := metav1.NewControllerRef(volumeManager, crv1.GVK)
	for _, handler := range h.dataHandlers {
		for _, vConfig := range volumeManager.Spec.VolumeConfigs {
			if handler.GetSourceType() == vConfig.SourceType {
				handler.OnDelete(volumeManager.Namespace(), vConfig, *controllerRef)
			}
		}
	}
}
