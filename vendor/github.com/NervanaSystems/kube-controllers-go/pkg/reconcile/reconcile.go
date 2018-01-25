package reconcile

import (
	"context"
	"fmt"
	"strings"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/NervanaSystems/kube-controllers-go/pkg/crd"
	"github.com/NervanaSystems/kube-controllers-go/pkg/resource"
	"github.com/NervanaSystems/kube-controllers-go/pkg/states"
	"github.com/golang/glog"
)

// Reconciler periodically checks the status of subresources and takes
// various self-healing and convergence actions. These include updating
// the top-level custom resource status, re-creating missing subresources,
// deleting orphaned subresources, et cetera.
//
// See the docs/reconciliation.md file for a detailed description of the
// reconciliation policy.
type Reconciler struct {
	namespace       string
	gvk             schema.GroupVersionKind
	crdHandle       *crd.Handle
	crdClient       crd.Client
	resourceClients []resource.Client
}

// New returns a new Reconciler.
func New(namespace string, gvk schema.GroupVersionKind, crdHandle *crd.Handle, crdClient crd.Client, resourceClients []resource.Client) *Reconciler {
	return &Reconciler{
		namespace:       namespace,
		gvk:             gvk,
		crdHandle:       crdHandle,
		crdClient:       crdClient,
		resourceClients: resourceClients,
	}
}

// Run starts the reconciliation loop and blocks until the context is done, or
// there is an unrecoverable error. Reconciliation actions are done at the
// supplied interval.
func (r *Reconciler) Run(ctx context.Context, interval time.Duration) error {
	glog.V(4).Infof("Starting reconciler for %v.%v.%v", r.gvk.Group, r.gvk.Version, r.gvk.Kind)
	go wait.Until(r.run, interval, ctx.Done())
	<-ctx.Done()
	return ctx.Err()
}

type subresource struct {
	client    resource.Client
	object    runtime.Object
	lifecycle lifecycle
}

type subresources []*subresource

// Contains subresources grouped by their controlling resource.
type subresourceMap map[string]subresources

type action struct {
	newCRState           states.State
	newCRReason          string
	subresourcesToCreate subresources
	subresourcesToDelete subresources
}

func (a action) String() string {
	var sCreateNames []string
	for _, s := range a.subresourcesToCreate {
		sCreateNames = append(sCreateNames, s.client.Plural())
	}
	var sDeleteNames []string
	for _, s := range a.subresourcesToDelete {
		sDeleteNames = append(sDeleteNames, s.client.Plural())
	}
	return fmt.Sprintf(
		`{
  newCRState: "%s",
  newCRReason: "%s",
  subresourcesToCreate: "%s",
  subresourcesToDelete: "%s"
}`,
		a.newCRState,
		a.newCRReason,
		strings.Join(sCreateNames, ", "),
		strings.Join(sDeleteNames, ", "))
}

func (r *Reconciler) run() {
	subresourcesByCR := r.groupSubresourcesByCustomResource()
	for crName, subs := range subresourcesByCR {
		a, cr, err := r.planAction(crName, subs)
		if err != nil {
			glog.Errorf(`failed to plan action for custom resource: [%s] subresources: [%v] error: [%s]`, crName, subresourcesByCR, err.Error())
			continue
		}
		glog.Infof("planned action: %s", a.String())
		errs := r.executeAction(crName, cr, a)
		if len(errs) > 0 {
			glog.Errorf(`failed to execute action for custom resource: [%s] subresources: %v errors: %v`, crName, subresourcesByCR, errs)
		}
	}
}

// TODO(CD): groupSubresourcesByCustomResource() doesn't work for a custom
// resource with no sub-resource(s) or the sub-resource have been deleted.
// As resourceClient.List() will not have any sub-resource belonging to the
// custom resource, result will not have the controller name as one of its
// keys.
//
// To fix the problem, we could do a List from the CR client and then iterate
// over those names instead of keys from the intermediate result map we built
// based on the subresources.
func (r *Reconciler) groupSubresourcesByCustomResource() subresourceMap {
	result := subresourceMap{}

	// Get the list of crs.
	crListObj, err := r.crdClient.List(r.namespace, map[string]string{})
	if err != nil || crListObj == nil {
		glog.Warningf("[reconcile] could not list custom resources. Got error %v %v", err, crListObj)
		return result
	}
	customResourceList := crListObj.(crd.CustomResourceList)

	// Get the list of custom resources
	crList := customResourceList.GetItems()
	// Return if the list is empty
	if len(crList) == 0 {
		glog.Warningf("[reconcile] custom resources list is empty")
		return result
	}

	for _, resourceClient := range r.resourceClients {
		objects, err := resourceClient.List(r.namespace, map[string]string{})
		if err != nil {
			glog.Warningf(`[reconcile] failed to list "%s" subresources`, resourceClient.Plural())
			continue
		}

		for _, obj := range objects {
			controllerRef := metav1.GetControllerOf(obj)
			if controllerRef == nil {
				glog.V(4).Infof("[reconcile] ignoring sub-resource %v, %v as it doesn not have a controller reference", obj.GetName(), r.namespace)
				continue
			}
			// Only manipulate controller-created subresources.
			if controllerRef.APIVersion != r.gvk.GroupVersion().String() || controllerRef.Kind != r.gvk.Kind {
				glog.V(4).Infof("[reconcile] ignoring sub-resource %v, %v as controlling custom resource is from a different group, version and kind", obj.GetName(), r.namespace)
				continue
			}

			subLifecycle := exists
			objMeta, err := meta.Accessor(obj)
			if err != nil {
				glog.Warningf("[reconcile] error getting meta accessor for subresource: %v", err)
				continue
			}
			if objMeta.GetDeletionTimestamp() != nil {
				subLifecycle = deleting
			}

			runtimeObj, ok := obj.(runtime.Object)
			if !ok {
				glog.Warningf("[reconcile] error asserting metav1.Object as runtime.Object: %v", err)
				continue
			}

			controllerName := controllerRef.Name
			objList := result[controllerName]
			result[controllerName] = append(objList, &subresource{resourceClient, runtimeObj, subLifecycle})
		}
	}

	// Iterate over the crs to get the list of missing sub resources
	// ASSUMPTION: There is at most one subresource of each kind per
	//             custom resource. We use the plural form as a key
	for _, item := range crList {
		cr, ok := item.(crd.CustomResource)
		if !ok {
			glog.Warningf("[reconcile] failed to assert item %v to type CustomResource", item)
			continue
		}

		subs, ok := result[cr.Name()]
		if !ok {
			glog.Warningf("[reconcile] no sub-resources found for cr %v", cr.Name())
		}

		// Find non-existing subresources based on the expected subresource clients.
		existingSubs := map[string]struct{}{}
		for _, sub := range subs {
			existingSubs[sub.client.Plural()] = struct{}{}
		}

		for _, subClient := range r.resourceClients {
			_, exists := existingSubs[subClient.Plural()]
			if !exists {
				result[cr.Name()] = append(subs, &subresource{subClient, nil, doesNotExist})
			}
		}

	}

	return result
}

func (subs subresources) filter(predicate func(s *subresource) bool) subresources {
	var result subresources
	for _, sub := range subs {
		if predicate(sub) {
			result = append(result, sub)
		}
	}
	return result
}

func (subs subresources) any(predicate func(s *subresource) bool) bool {
	return len(subs.filter(predicate)) > 0
}

func (subs subresources) all(predicate func(s *subresource) bool) bool {
	return len(subs.filter(predicate)) == len(subs)
}

func (r *Reconciler) planAction(controllerName string, subs subresources) (*action, crd.CustomResource, error) {
	// If the controller name is empty, these are not our subresources;
	// do nothing.
	if controllerName == "" {
		return &action{}, nil, nil
	}

	// Compute the current lifecycle phase of the custom resource.
	customResourceLifecycle := exists
	crObj, err := r.crdClient.Get(r.namespace, controllerName)
	if err != nil && apierrors.IsNotFound(err) {
		customResourceLifecycle = doesNotExist
	}
	crMeta, err := meta.Accessor(crObj)
	if err != nil {
		glog.Warningf("[reconcile] error getting meta accessor for controlling custom resource: %v", err)
	} else if crMeta.GetDeletionTimestamp() != nil {
		customResourceLifecycle = deleting
	}

	// If the custom resource is deleting or does not exist, clean up all
	// subresources.
	if customResourceLifecycle.isOneOf(doesNotExist, deleting) {
		return &action{subresourcesToDelete: subs}, nil, nil
	}

	cr, ok := crObj.(crd.CustomResource)
	if !ok {
		return &action{}, nil, fmt.Errorf("object retrieved from CRD client not an instance of crd.CustomResource: [%v]", crObj)
	}

	customResourceSpecState := cr.GetSpecState()
	customResourceStatusState := cr.GetStatusState()

	// If the desired custom resource state is running or completed AND
	// the custom resource is in a terminal state, then delete all subresources.
	if customResourceSpecState.IsOneOf(states.Running, states.Completed) &&
		customResourceStatusState.IsOneOf(states.Completed, states.Failed) {
		return &action{subresourcesToDelete: subs}, nil, nil
	}

	// If the desired custom resource state is running or completed AND
	// the current custom resource status is non-terminal, ANY non-ephemeral
	// subresource that is failed, does not exist or has been deleted causes
	// the custom resource current state to move to failed.
	if customResourceSpecState.IsOneOf(states.Running, states.Completed) &&
		customResourceStatusState.IsOneOf(states.Pending, states.Running) {
		if subs.any(func(s *subresource) bool {
			return !s.client.IsEphemeral() &&
				s.lifecycle.isOneOf(doesNotExist, deleting) ||
				s.client.GetStatusState(s.object) == states.Failed
		}) {
			// Set CR to failed
			return &action{
				newCRState: states.Failed,
			}, cr, nil
		}
	}

	// If the desired custom resource state is completed AND
	// the current custom resource status is pending or running, then if ANY
	// subresource is completed, set the current custom resource state to
	// completed.
	if customResourceSpecState == states.Completed && customResourceStatusState.IsOneOf(states.Pending, states.Running) {
		if subs.any(func(s *subresource) bool {
			return s.client.GetStatusState(s.object) == states.Completed
		}) {
			// Set CR as completed
			return &action{
				newCRState: states.Completed,
			}, cr, nil
		}
	}

	// If the desired custom resource state is running or completed AND
	// the current custom resource state is pending or running, then
	// re-create any nonexisting ephemeral subresources.
	if customResourceSpecState.IsOneOf(states.Running, states.Completed) &&
		customResourceStatusState.IsOneOf(states.Pending, states.Running) {
		toRecreate := subs.filter(func(s *subresource) bool {
			return s.client.IsEphemeral() &&
				(s.lifecycle == exists && s.client.GetStatusState(s.object) == states.Failed ||
					s.lifecycle == doesNotExist)
		})

		if len(toRecreate) > 0 {
			// Recreate
			return &action{subresourcesToCreate: toRecreate}, cr, nil
		}
	}

	// If the desired custom resource state is running or completed AND
	// the current custom resource state is running AND
	// ANY subresource is pending, then set the current custom resource state
	// to pending.
	if customResourceSpecState.IsOneOf(states.Running, states.Completed) &&
		customResourceStatusState == states.Running {
		if subs.any(func(s *subresource) bool {
			return s.client.GetStatusState(s.object) == states.Pending
		}) {
			// Set CR as pending
			return &action{
				newCRState: states.Pending,
			}, cr, nil
		}
	}

	// If the desired custom resource state is running or completed AND
	// the current custom resource state is pending AND
	// ALL subresources are running, then set the current custom resource state
	// to running.
	if customResourceSpecState.IsOneOf(states.Running, states.Completed) &&
		customResourceStatusState == states.Pending {
		// All resources must be running for us to consider the custom resource as running.
		if subs.all(func(s *subresource) bool {
			return s.client.GetStatusState(s.object) == states.Running
		}) {
			// Set CR as running
			return &action{
				newCRState: states.Running,
			}, cr, nil
		}
	}

	// Default case: do nothing.
	return &action{}, cr, nil
}

func (r *Reconciler) executeAction(controllerName string, cr crd.CustomResource, a *action) []error {
	errors := []error{}

	glog.V(4).Infof(`executing reconcile action for "%s" resource "%s" in namespace "%s"`, r.crdHandle.Plural, controllerName, r.namespace)
	if a.newCRState != "" {
		glog.Infof(`updating "%s" custom resource for controller "%s" in namespace "%s"`, r.crdHandle.Plural, controllerName, r.namespace)
		cr.SetStatusStateWithMessage(a.newCRState, a.newCRReason)
		_, err := r.crdClient.Update(cr)
		if err != nil {
			glog.Errorf(`error updating custom resource state for "%s" in namespace "%s"`, controllerName, r.namespace)
			errors = append(errors, err)
		}
	}

	for _, s := range a.subresourcesToCreate {
		glog.Infof(`creating "%s" subresource for controller "%s" in namespace "%s"`, s.client.Plural(), controllerName, r.namespace)
		err := s.client.Create(r.namespace, cr)
		if err != nil {
			glog.Errorf(`error creating "%s" subresource for controller "%s" in namespace "%s"`, s.client.Plural(), controllerName, r.namespace)
			errors = append(errors, err)
		}
	}

	for _, s := range a.subresourcesToDelete {
		glog.Infof(`deleting "%s" subresource for controller "%s" in namespace "%s"`, s.client.Plural(), controllerName, r.namespace)
		err := s.client.Delete(r.namespace, controllerName)
		if err != nil {
			glog.Errorf(`error deleting "%s" subresource for controller "%s" in namespace "%s"`, s.client.Plural(), controllerName, r.namespace)
			errors = append(errors, err)
		}
	}

	return errors
}
