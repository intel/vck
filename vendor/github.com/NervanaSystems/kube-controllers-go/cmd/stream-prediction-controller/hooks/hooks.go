package hooks

import (
	"fmt"

	"github.com/golang/glog"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	crv1 "github.com/NervanaSystems/kube-controllers-go/cmd/stream-prediction-controller/apis/cr/v1"
	"github.com/NervanaSystems/kube-controllers-go/pkg/crd"
	"github.com/NervanaSystems/kube-controllers-go/pkg/resource"
	"github.com/NervanaSystems/kube-controllers-go/pkg/states"
)

// StreamPredictionHooks implements controller.Hooks interface
type StreamPredictionHooks struct {
	resourceClients []resource.Client
	crdClient       crd.Client
}

// NewStreamPredictionHooks creates and returns a new instance of the StreamPredictionHooks
func NewStreamPredictionHooks(crdClient crd.Client, resourceClients []resource.Client) *StreamPredictionHooks {
	return &StreamPredictionHooks{
		resourceClients: resourceClients,
		crdClient:       crdClient,
	}
}

// Add handles the addition of a new stream prediction object
func (h *StreamPredictionHooks) Add(obj interface{}) {
	streamCrd, ok := obj.(*crv1.StreamPrediction)
	if !ok {
		glog.Errorf("object received is not of type StreamPrediction %v", obj)
		return
	}
	glog.Infof("add called, got CRD: %v", streamCrd)

	streamPredict := streamCrd.DeepCopy()

	// If created with a terminal desired state. We immediately change the stream prediction into that status.
	if states.IsTerminal(streamCrd.GetSpecState()) {
		streamPredict.Status = crv1.StreamPredictionStatus{
			State:   streamCrd.Spec.State,
			Message: "Added. Detected in desired terminal state and controller marked stream prediction as " + string(streamCrd.Spec.State),
		}

		h.crdClient.Update(streamPredict)
		return
	}

	// Upon receipt of a new SP CR, we mark its status as `Pending'.
	streamPredict.Status = crv1.StreamPredictionStatus{
		State:   states.Pending,
		Message: "Added. Beginning sub-resource deployment",
	}

	obj, err := h.crdClient.Update(streamPredict)
	if err != nil {
		fmt.Println(err)
		glog.Warningf("error updating status for stream prediction %s: %v\n",
			streamPredict.Spec.StreamDataSpec.StreamName, err)
		return
	}

	// obj is the custom resource after the update happened i.e. contains the most recent
	// resource version and just needs to be type cast back to a stream prediction object.
	streamCrd, ok = obj.(*crv1.StreamPrediction)
	if !ok {
		glog.Errorf("object received is not of type StreamPrediction %v", obj)
		return
	}
	streamPredict = streamCrd.DeepCopy()

	glog.Infof("status updated for stream prediction %s:",
		streamPredict.Spec.StreamDataSpec.StreamName,
		streamPredict.Status.State)

	// Next, we create its requisite sub-resources.
	// If this creation fails, we mark the SP to be in an error state.
	// We don't delete sub-resources here, as the Update handles a `Failed'
	// status.  This is due to there being multiple writers to a SP's status.
	// E.g., the garbage collector / reconciler could also set a CR's status to
	// `Failed'.
	err = h.addResources(streamPredict)
	if err != nil {
		streamPredict.Status = crv1.StreamPredictionStatus{
			State:   states.Failed,
			Message: "Failed to deploy sub-resources",
		}
		_, err := h.crdClient.Update(streamPredict)
		if err != nil {
			glog.Warningf(
				"error updating status for stream prediction %s: %v\n",
				streamPredict.Spec.StreamDataSpec.StreamName, err)
			return
		}
		return
	}

	streamPredict.Status = crv1.StreamPredictionStatus{
		State:   states.Running,
		Message: "Running sub-resources",
	}
	_, err = h.crdClient.Update(streamPredict)
	if err != nil {
		glog.Warningf("error updating status: %s\n", err)
		return
	}

	glog.Infof("updated status: %s\n", streamPredict.Status.State)
}

// Update handles the update of a stream prediction object
func (h *StreamPredictionHooks) Update(_, newObj interface{}) {
	newStreamPredict, ok := newObj.(*crv1.StreamPrediction)
	if !ok {
		glog.Errorf("object received is not of type StreamPrediction %v",
			newObj)
		return
	}

	// If the SP CR's spec has been updated to `Completed', then we delete
	// subresources, and mark it as `Completed' in its status.
	if newStreamPredict.Spec.State == states.Completed {
		glog.Infof(
			"stream prediction %s has been marked for undeployment",
			newStreamPredict)
		h.deleteResources(newStreamPredict)
		newStreamPredict.Status = crv1.StreamPredictionStatus{
			State:   states.Completed,
			Message: "Stream Prediction completed",
		}
		if _, err := h.crdClient.Update(newStreamPredict); err != nil {
			glog.Warningf("error updating status: %v\n", err)
			return
		}
		glog.Infof("Successfully deleted subresources")
		return
	}

	// If the SP CR has been marked to be in an `Failed' state, either by the
	// sub-resource reconciler or during creation, we delete its sub-resources.
	if newStreamPredict.Status.State == states.Failed {
		glog.Infof("stream prediction %s is in an error state, "+
			"deleting subresources",
			newStreamPredict.Spec.StreamDataSpec.StreamName)
		h.deleteResources(newStreamPredict)
		glog.Info("Successfully deleted subresources")
		return
	}
}

// Delete handles the deletion of a stream prediction object
func (h *StreamPredictionHooks) Delete(obj interface{}) {
	streamPredict, ok := obj.(*crv1.StreamPrediction)
	if !ok {
		glog.Errorf("object received is not of type StreamPrediction %v", obj)
		return
	}
	glog.Infof("delete, got crd: %s", streamPredict.SelfLink)

	//Delete the resources using name for now.
	h.deleteResources(streamPredict)
}

func (h *StreamPredictionHooks) addResources(streamPredict *crv1.StreamPrediction) error {
	// Add controller reference.
	// See https://kubernetes.io/docs/concepts/workloads/controllers/garbage-collection/
	// for more details on owner references.
	ownerRef := metav1.NewControllerRef(streamPredict, crv1.GVK)

	for _, resourceClient := range h.resourceClients {
		err := resourceClient.Create(streamPredict.Namespace(), struct {
			*crv1.StreamPrediction
			metav1.OwnerReference
		}{
			streamPredict,
			*ownerRef,
		})
		if err != nil {
			glog.Errorf("received err: %v while creating object", err)
			return err
		}
	}
	glog.Infof("resource creation complete for stream prediction \"%s\"", streamPredict.Name())
	return nil
}

func (h *StreamPredictionHooks) deleteResources(streamPredict *crv1.StreamPrediction) {
	for _, resourceClient := range h.resourceClients {
		if err := resourceClient.Delete(streamPredict.Namespace(), streamPredict.Name()); err != nil {
			glog.Errorf("resource deletion failed for stream prediction \"%s\": %v", streamPredict.Name(), err)
		}
	}
	glog.Info("resource deletion complete for stream prediction \"%s\"", streamPredict.Name())
}
