package hooks

import (
	"fmt"

	"github.com/golang/glog"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	vckv1 "github.com/ppkube/vck/pkg/apis/vck/v1"
	vckv1_volume_manager "github.com/ppkube/vck/pkg/client/clientset/versioned/typed/vck/v1"
	"github.com/ppkube/vck/pkg/handlers"
	"github.com/ppkube/vck/pkg/states"
)

// VolumeManagerHooks implements controller.Hooks interface
type VolumeManagerHooks struct {
	crdClient    vckv1_volume_manager.VolumeManagerInterface
	dataHandlers []handlers.DataHandler
}

// NewVolumeManagerHooks creates and returns a new instance of the VolumeManagerHooks
func NewVolumeManagerHooks(crdClient vckv1_volume_manager.VolumeManagerInterface, dataHandlers []handlers.DataHandler) *VolumeManagerHooks {
	return &VolumeManagerHooks{
		crdClient:    crdClient,
		dataHandlers: dataHandlers,
	}
}

func (h *VolumeManagerHooks) add(obj interface{}) {
	volumeManager, ok := obj.(*vckv1.VolumeManager)
	if !ok {
		glog.Errorf("object received is not of type VolumeManager %v", obj)
		return
	}
	glog.V(4).Infof("Volume Manager add hook - got: %v", volumeManager)

	volumeManagerCopy := volumeManager.DeepCopy()

	// If created with a Failed desired state. We immediately change the volume
	// manager status to Failed.
	if volumeManagerCopy.Spec.State == states.Failed {
		volumeManagerCopy.Status = vckv1.VolumeManagerStatus{
			State:   volumeManagerCopy.Spec.State,
			Message: "Added with desired state as failed and controller marked volume manager as " + string(volumeManagerCopy.Spec.State),
		}

		h.crdClient.Update(volumeManagerCopy)
		return
	}

	// Mark the CR as pending before starting to invoke the handlers.
	volumeManagerCopy.Status = vckv1.VolumeManagerStatus{
		State:   states.Pending,
		Message: fmt.Sprintf("Beginning sub-resource deployment"),
	}

	volumeManagerCopy, err := h.crdClient.Update(volumeManagerCopy)
	if err != nil {
		glog.Warningf("error updating status for volume manager %s: %v\n", volumeManagerCopy.Name, err)
		return
	}

	controllerRef := metav1.NewControllerRef(volumeManagerCopy, vckv1.GVK)

	vStatuses := []vckv1.Volume{}
	for _, handler := range h.dataHandlers {
		for _, vConfig := range volumeManagerCopy.Spec.VolumeConfigs {
			if handler.GetSourceType() == vConfig.SourceType {
				vStatus := handler.OnAdd(volumeManagerCopy.Namespace, vConfig, *controllerRef)
				vStatuses = append(vStatuses, vStatus)
			}
		}
	}

	for _, vStatus := range vStatuses {
		// If any of the volume claim was not successful, mark the CR as Failed.
		if vStatus.Message != vckv1.SuccessfulVolumeStatusMessage {
			volumeManagerCopy.Status = vckv1.VolumeManagerStatus{
				Volumes: vStatuses,
				State:   states.Failed,
				Message: fmt.Sprintf("failed to deploy all the sub-resources"),
			}

			_, err := h.crdClient.Update(volumeManagerCopy)
			if err != nil {
				glog.Warningf("error updating status for volume manager %s: %v\n", volumeManagerCopy.Name, err)
				return
			}
		}
	}

	// Mark the CR as Running.
	volumeManagerCopy.Status = vckv1.VolumeManagerStatus{
		Volumes: vStatuses,
		State:   states.Running,
		Message: fmt.Sprintf("successfully deployed all sub-resources"),
	}

	_, err = h.crdClient.Update(volumeManagerCopy)
	if err != nil {
		glog.Warningf("error updating status for volume manager %s: %v\n", volumeManagerCopy.Name, err)
		return
	}
}

// Add handles the addition of a new volume manager object
func (h *VolumeManagerHooks) Add(obj interface{}) {
	// A goroutine is spawned to handle the addition of a CR.
	// This work-around is required as we wait for completion of sub-resource(s)
	// for some data handlers before moving the custom resoure from Pending to
	// Running. Ideally, this case should be handled in a reconciler or the
	// controller should be modified to handle multiple additions simultaneously.
	// See https://github.com/kubernetes/sample-controller/blob/master/controller.go#L171-L173
	// for example.
	// TODO(balajismaniam): Remove this goroutine.
	go h.add(obj)
}

// Update handles the update of a volume manager object
func (h *VolumeManagerHooks) Update(oldObj, newObj interface{}) {
	newVolumeManager, ok := newObj.(*vckv1.VolumeManager)
	if !ok {
		glog.Errorf("object received is not of type VolumeManager %v", newObj)
		return
	}

	oldVolumeManager, ok := oldObj.(*vckv1.VolumeManager)
	if !ok {
		glog.Errorf("object received is not of type VolumeManager %v", oldObj)
		return
	}
	glog.V(4).Infof("Volume Manager update hook - got old: %v new: %v", oldVolumeManager, newVolumeManager)

	controllerRef := metav1.NewControllerRef(newVolumeManager, vckv1.GVK)
	// Delete all the sub-resources if the CR is in a failed state.
	if newVolumeManager.Status.State == states.Failed {
		// Delete all the sub-resources.
		for _, handler := range h.dataHandlers {
			for idx, vConfig := range newVolumeManager.Spec.VolumeConfigs {
				if handler.GetSourceType() == vConfig.SourceType && len(newVolumeManager.Status.Volumes) > idx {
					handler.OnDelete(newVolumeManager.Namespace, vConfig, newVolumeManager.Status.Volumes[idx], *controllerRef)
				}
			}
		}
	}
}

// Delete handles the deletion of a volume manager object
func (h *VolumeManagerHooks) Delete(obj interface{}) {
	volumeManager, ok := obj.(*vckv1.VolumeManager)
	if !ok {
		glog.Errorf("object received is not of type VolumeManager %v", obj)
		return
	}
	glog.V(4).Infof("Volume Manager delete hook - got: %v", volumeManager)

	controllerRef := metav1.NewControllerRef(volumeManager, vckv1.GVK)
	for _, handler := range h.dataHandlers {
		for idx, vConfig := range volumeManager.Spec.VolumeConfigs {
			if handler.GetSourceType() == vConfig.SourceType && len(volumeManager.Status.Volumes) > idx {
				handler.OnDelete(volumeManager.Namespace, vConfig, volumeManager.Status.Volumes[idx], *controllerRef)
			}
		}
	}
}
