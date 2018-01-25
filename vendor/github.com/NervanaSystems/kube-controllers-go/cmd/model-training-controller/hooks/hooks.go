package hooks

import (
	"fmt"

	"github.com/golang/glog"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	crv1 "github.com/NervanaSystems/kube-controllers-go/cmd/model-training-controller/apis/cr/v1"
	"github.com/NervanaSystems/kube-controllers-go/pkg/crd"
	"github.com/NervanaSystems/kube-controllers-go/pkg/resource"
	"github.com/NervanaSystems/kube-controllers-go/pkg/states"
)

// ModelTrainingHooks implements controller.Hooks interface
type ModelTrainingHooks struct {
	resourceClients []resource.Client
	crdClient       crd.Client
}

// NewModelTrainingHooks creates and returns a new instance of the ModelTrainingHooks
func NewModelTrainingHooks(crdClient crd.Client, resourceClients []resource.Client) *ModelTrainingHooks {
	return &ModelTrainingHooks{
		resourceClients: resourceClients,
		crdClient:       crdClient,
	}
}

// Add handles the addition of a new model training object
func (h *ModelTrainingHooks) Add(obj interface{}) {
	modelTrainingCrd, ok := obj.(*crv1.ModelTraining)
	if !ok {
		glog.Errorf("object received is not of type ModelTraining %v", obj)
		return
	}
	glog.V(4).Infof("Model Training add hook - got: %v", modelTrainingCrd)

	modelTrain := modelTrainingCrd.DeepCopy()

	// If created with a Failed desired state. We immediately change the model training into that status.
	if modelTrain.Spec.State == states.Failed {
		modelTrain.Status = crv1.ModelTrainingStatus{
			State:   modelTrainingCrd.Spec.State,
			Message: "Added. Detected in desired terminal state and controller marked model training as " + string(modelTrainingCrd.Spec.State),
		}

		h.crdClient.Update(modelTrain)
		return
	}

	// Upon receipt of a new model training CR, we mark its status as `Pending'.
	modelTrain.Status = crv1.ModelTrainingStatus{
		State:   states.Pending,
		Message: "Added. Beginning sub-resource deployment",
	}

	obj, err := h.crdClient.Update(modelTrain)
	if err != nil {
		fmt.Println(err)
		glog.Warningf("error updating status for model training %s: %v\n",
			modelTrain.ObjectMeta.Name, err)
		return
	}

	// obj is the custom resource after the update happened i.e. contains the most recent
	// resource version and just needs to be type cast back to a model training object.
	modelTrainingCrd, ok = obj.(*crv1.ModelTraining)
	if !ok {
		glog.Errorf("object received is not of type ModelTraining %v", obj)
		return
	}
	modelTrain = modelTrainingCrd.DeepCopy()

	glog.Infof("status updated for model training %s:",
		modelTrain.ObjectMeta.Name,
		modelTrain.Status.State)

	// Next, we create its requisite sub-resources.
	// If this creation fails, we mark the CR to be in an error state.
	// We don't delete sub-resources here, as the Update handles an `Failed'
	// status.  This is due to there being multiple writers to a CR's status.
	// E.g., the garbage collector / reconciler could also set a CR's status to
	// `Failed'.
	err = h.addResources(modelTrain)
	if err != nil {
		modelTrain.Status = crv1.ModelTrainingStatus{
			State:   states.Failed,
			Message: "Failed to deploy sub-resources",
		}
		_, err := h.crdClient.Update(modelTrain)
		if err != nil {
			glog.Warningf(
				"error updating status for model training %s: %v\n",
				modelTrain.ObjectMeta.Name, err)
			return
		}
		return
	}

	modelTrain.Status = crv1.ModelTrainingStatus{
		State:   states.Running,
		Message: "Sub-resources have been deployed",
	}
	_, err = h.crdClient.Update(modelTrain)
	if err != nil {
		glog.Warningf("error updating status: %s\n", err)
		return
	}

	glog.Infof("updated status: %s\n", modelTrain.Status.State)
}

// Update handles the update of a model training object
func (h *ModelTrainingHooks) Update(oldObj, newObj interface{}) {
	newModelTraining, ok := newObj.(*crv1.ModelTraining)
	if !ok {
		glog.Errorf("object received is not of type ModelTraining %v", newObj)
		return
	}

	oldModelTraining, ok := oldObj.(*crv1.ModelTraining)
	if !ok {
		glog.Errorf("object received is not of type ModelTraining %v", oldObj)
		return
	}

	glog.V(4).Infof("Model Training update hook - got old: %v new: %v", oldModelTraining, newModelTraining)

	// If the CR has been marked to be in an Failed state, either by the
	// sub-resource reconciler or during creation, we delete its sub-resources.
	if newModelTraining.Status.State == states.Failed {
		glog.Infof("model training %s is in a Failed state, "+
			"deleting subresources",
			newModelTraining.ObjectMeta.Name)
		h.deleteResources(newModelTraining)
		glog.Infof("Successfully deleted subresources for model training %s", newModelTraining)
		return
	}
}

// Delete handles the deletion of a model training object
func (h *ModelTrainingHooks) Delete(obj interface{}) {
	modelTrain, ok := obj.(*crv1.ModelTraining)
	if !ok {
		glog.Errorf("object received is not of type ModelTraining %v", obj)
		return
	}
	glog.V(4).Infof("Model Training add hook - got: %v", modelTrain)

	//Delete the resources using name for now.
	h.deleteResources(modelTrain)
}

func (h *ModelTrainingHooks) addResources(modelTrain *crv1.ModelTraining) error {
	// Add controller reference.
	// See https://kubernetes.io/docs/concepts/workloads/controllers/garbage-collection/
	// for more details on owner references.
	ownerRef := metav1.NewControllerRef(modelTrain, crv1.GVK)

	for _, resourceClient := range h.resourceClients {
		err := resourceClient.Create(modelTrain.Namespace(), struct {
			*crv1.ModelTraining
			metav1.OwnerReference
		}{
			modelTrain,
			*ownerRef,
		})
		if err != nil {
			glog.Errorf("received err: %v while creating object", err)
			return err
		}
	}
	glog.Infof("resource creation complete for model training \"%s\"", modelTrain.Name())
	return nil
}

func (h *ModelTrainingHooks) deleteResources(modelTrain *crv1.ModelTraining) {
	for _, resourceClient := range h.resourceClients {
		if err := resourceClient.Delete(modelTrain.Namespace(), modelTrain.Name()); err != nil {
			glog.Errorf("resource deletion failed for model training \"%s\": %v", modelTrain.Name(), err)
		}
	}
	glog.Info("resource deletion complete for model training \"%s\"", modelTrain.Name())
}
