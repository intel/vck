package hooks

import (
	"github.com/golang/glog"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	crv1 "github.com/NervanaSystems/kube-controllers-go/cmd/batch-prediction-controller/apis/cr/v1"
	"github.com/NervanaSystems/kube-controllers-go/pkg/crd"
	"github.com/NervanaSystems/kube-controllers-go/pkg/resource"
)

// BatchPredictionHooks implements controller.Hooks interface
type BatchPredictionHooks struct {
	resourceClients []resource.Client
	crdClient       crd.Client
}

// NewBatchPredictionHooks creates and returns a new instance of the BatchPredictionHooks
func NewBatchPredictionHooks(crdClient crd.Client, resourceClients []resource.Client) *BatchPredictionHooks {
	return &BatchPredictionHooks{
		resourceClients: resourceClients,
		crdClient:       crdClient,
	}
}

// Add handles the addition of a new batch prediction object
func (h *BatchPredictionHooks) Add(obj interface{}) {
	batchPredictionCrd, ok := obj.(*crv1.BatchPrediction)
	if !ok {
		glog.Errorf("object received is not of type BatchPrediction %v", obj)
		return
	}
	glog.V(4).Infof("Batch Prediction add hook - got: %v", batchPredictionCrd)
}

// Update handles the update of a batch prediction object
func (h *BatchPredictionHooks) Update(oldObj, newObj interface{}) {
	newBatchPrediction, ok := newObj.(*crv1.BatchPrediction)
	if !ok {
		glog.Errorf("object received is not of type BatchPrediction %v", newObj)
		return
	}

	oldBatchPrediction, ok := oldObj.(*crv1.BatchPrediction)
	if !ok {
		glog.Errorf("object received is not of type BatchPrediction %v", oldObj)
		return
	}

	glog.V(4).Infof("Batch Prediction update hook - got old: %v new: %v", oldBatchPrediction, newBatchPrediction)
}

// Delete handles the deletion of a batch prediction object
func (h *BatchPredictionHooks) Delete(obj interface{}) {
	batchPredict, ok := obj.(*crv1.BatchPrediction)
	if !ok {
		glog.Errorf("object received is not of type BatchPrediction %v", obj)
		return
	}
	glog.V(4).Infof("Batch Prediction add hook - got: %v", batchPredict)

	//Delete the resources using name for now.
	h.deleteResources(batchPredict)
}

func (h *BatchPredictionHooks) addResources(batchPredict *crv1.BatchPrediction) error {
	// Add controller reference.
	// See https://kubernetes.io/docs/concepts/workloads/controllers/garbage-collection/
	// for more details on owner references.
	ownerRef := metav1.NewControllerRef(batchPredict, crv1.GVK)

	for _, resourceClient := range h.resourceClients {
		err := resourceClient.Create(batchPredict.Namespace(), struct {
			*crv1.BatchPrediction
			metav1.OwnerReference
		}{
			batchPredict,
			*ownerRef,
		})
		if err != nil {
			glog.Errorf("received err: %v while creating object", err)
			return err
		}
	}
	glog.Infof("resource creation complete for batch prediction \"%s\"", batchPredict.Name())
	return nil
}

func (h *BatchPredictionHooks) deleteResources(batchPredict *crv1.BatchPrediction) {
	for _, resourceClient := range h.resourceClients {
		if err := resourceClient.Delete(batchPredict.Namespace(), batchPredict.Name()); err != nil {
			glog.Errorf("resource deletion failed for batch prediction \"%s\": %v", batchPredict.Name(), err)
		}
	}
	glog.Info("resource deletion complete for batch prediction \"%s\"", batchPredict.Name())
}
