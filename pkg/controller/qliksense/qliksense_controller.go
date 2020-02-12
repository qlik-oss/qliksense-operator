package qliksense

import (
	"context"
	_ "fmt"
	"github.com/go-logr/logr"
	qlikv1 "github.com/qlik-oss/qliksense-operator/pkg/apis/qlik/v1"
	_ "gopkg.in/yaml.v2"
	appsv1 "k8s.io/api/apps/v1"
	batch_v1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	_ "k8s.io/apimachinery/pkg/types"
	_ "k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_qliksense")

const (
	qliksenseFinalizer = "finalizer.qliksense.qlik.com"
	searchingLabel     = "release"
)

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new Qliksense Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileQliksense{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("qliksense-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Qliksense
	err = c.Watch(&source.Kind{Type: &qlikv1.Qliksense{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner Qliksense
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &qlikv1.Qliksense{},
	})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileQliksense implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileQliksense{}

// ReconcileQliksense reconciles a Qliksense object
type ReconcileQliksense struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a Qliksense object and makes changes based on the state read
// and what is in the Qliksense.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileQliksense) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling Qliksense")
	// Fetch the Qliksense instance
	instance := &qlikv1.Qliksense{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}
	/*
		// keep this for debugging pupose
		if b, err := yaml.Marshal(instance); err != nil {
			reqLogger.Error(err, "cannot marshal qliksense CR")
		} else {
			fmt.Println(string(b))
		}
	*/
	if err := r.updateServiceOwner(instance); err != nil {
		reqLogger.Error(err, "cannot update service owner")
		return reconcile.Result{}, err
	}
	if err := r.updateDeploymentOwner(instance); err != nil {
		reqLogger.Error(err, "cannot update deployments owner")
		return reconcile.Result{}, err
	}
	if err := r.updateStatefulSetOwner(instance); err != nil {
		reqLogger.Error(err, "cannot update deployments owner")
		return reconcile.Result{}, err
	}
	if err := r.updateConfigMapOwner(instance); err != nil {
		reqLogger.Error(err, "cannot update config map owner")
		return reconcile.Result{}, err
	}
	if err := r.updateSecretsOwner(instance); err != nil {
		reqLogger.Error(err, "cannot update secrets owner")
		return reconcile.Result{}, err
	}
	if err := r.updatePvcOwner(instance); err != nil {
		reqLogger.Error(err, "cannot update pvc owner")
		return reconcile.Result{}, err
	}
	if err := r.updateCronJobOwner(instance); err != nil {
		reqLogger.Error(err, "cannot update cronjob owner")
		return reconcile.Result{}, err
	}

	if err := r.updateCronJobOwner(instance); err != nil {
		reqLogger.Error(err, "cannot update cronjob owner")
		return reconcile.Result{}, err
	}
	if err := r.updateServiceAccountOwner(instance); err != nil {
		reqLogger.Error(err, "cannot update service account owner")
		return reconcile.Result{}, err
	}
	if err := r.updateRoleOwner(instance); err != nil {
		reqLogger.Error(err, "cannot update role owner")
		return reconcile.Result{}, err
	}
	if err := r.updateRoleBindingOwner(instance); err != nil {
		reqLogger.Error(err, "cannot update role binding owner")
		return reconcile.Result{}, err
	}
	if err := r.updateNetworkPolicyOwner(instance); err != nil {
		reqLogger.Error(err, "cannot update network policy owner")
		return reconcile.Result{}, err
	}
	if err := r.updateEngineOwner(instance); err != nil {
		reqLogger.Error(err, "cannot update Engine owner")
		return reconcile.Result{}, err
	}

	// Check if the qliksense instance is marked to be deleted, which is
	// indicated by the deletion timestamp being set.
	/*isQliksenseMarkedToBeDeleted := instance.GetDeletionTimestamp() != nil
	if isQliksenseMarkedToBeDeleted {
		if contains(instance.GetFinalizers(), qliksenseFinalizer) {
			// Run finalization logic for qliksenseFinalizer. If the
			// finalization logic fails, don't remove the finalizer so
			// that we can retry during the next reconciliation.
			if err := r.finalizeQliksense(reqLogger, instance); err != nil {
				return reconcile.Result{}, err
			}

			// Remove qliksenseFinalizer. Once all finalizers have been
			// removed, the object will be deleted.
			instance.SetFinalizers(remove(instance.GetFinalizers(), qliksenseFinalizer))
			err := r.client.Update(context.TODO(), instance)
			if err != nil {
				return reconcile.Result{}, err
			}
		}
		return reconcile.Result{}, nil
	}
	*/
	// Add finalizer for this CR
	/*if !contains(instance.GetFinalizers(), qliksenseFinalizer) {
		if err := r.addFinalizer(reqLogger, instance); err != nil {
			return reconcile.Result{}, err
		}
	}
	*/
	return reconcile.Result{}, nil
}

func (r *ReconcileQliksense) finalizeQliksense(reqLogger logr.Logger, qlik *qlikv1.Qliksense) error {
	// TODO(user): Add the cleanup steps that the operator
	// needs to do before the CR can be deleted. Examples
	// of finalizers include performing backups and deleting
	// resources that are not owned by this CR, like a PVC.
	if err := r.client.DeleteAllOf(context.TODO(), &corev1.Service{}, client.MatchingLabels{searchingLabel: qlik.Name}); err != nil {
		reqLogger.Error(err, "Cannot delete service")
		return nil
	}
	reqLogger.Info("Successfully finalized memcached")
	return nil
}

func (r *ReconcileQliksense) addFinalizer(reqLogger logr.Logger, m *qlikv1.Qliksense) error {
	reqLogger.Info("Adding Finalizer for the Memcached")
	m.SetFinalizers(append(m.GetFinalizers(), qliksenseFinalizer))

	// Update CR
	err := r.client.Update(context.TODO(), m)
	if err != nil {
		reqLogger.Error(err, "Failed to update Memcached with finalizer")
		return err
	}
	return nil
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func remove(list []string, s string) []string {
	for i, v := range list {
		if v == s {
			list = append(list[:i], list[i+1:]...)
		}
	}
	return list
}

func (r *ReconcileQliksense) updateServiceOwner(q *qlikv1.Qliksense) error {

	listObj := &corev1.ServiceList{}
	if err := r.client.List(context.TODO(), listObj, client.MatchingLabels{searchingLabel: q.Name}); err != nil {
		return err
	}
	for _, svc := range listObj.Items {
		//fmt.Println(svc.Name)
		if err := controllerutil.SetControllerReference(q, &svc, r.scheme); err != nil {
			return err
		}
		if err := r.client.Update(context.TODO(), &svc); err != nil {
			return err
		}
	}
	return nil
}

func (r *ReconcileQliksense) updateDeploymentOwner(q *qlikv1.Qliksense) error {

	listObj := &appsv1.DeploymentList{}
	if err := r.client.List(context.TODO(), listObj, client.MatchingLabels{searchingLabel: q.Name}); err != nil {
		return err
	}
	for _, dep := range listObj.Items {
		//fmt.Println(svc.Name)
		if err := controllerutil.SetControllerReference(q, &dep, r.scheme); err != nil {
			return err
		}
		if err := r.client.Update(context.TODO(), &dep); err != nil {
			return err
		}
	}
	return nil
}

func (r *ReconcileQliksense) updateStatefulSetOwner(q *qlikv1.Qliksense) error {

	listObj := &appsv1.StatefulSetList{}
	if err := r.client.List(context.TODO(), listObj, client.MatchingLabels{searchingLabel: q.Name}); err != nil {
		return err
	}
	for _, dep := range listObj.Items {
		//fmt.Println(svc.Name)
		if err := controllerutil.SetControllerReference(q, &dep, r.scheme); err != nil {
			return err
		}
		if err := r.client.Update(context.TODO(), &dep); err != nil {
			return err
		}
	}
	return nil
}

func (r *ReconcileQliksense) updateIngressOwner(q *qlikv1.Qliksense) error {

	listObj := &v1beta1.IngressList{}
	if err := r.client.List(context.TODO(), listObj, client.MatchingLabels{searchingLabel: q.Name}); err != nil {
		return err
	}
	for _, ing := range listObj.Items {
		//fmt.Println(svc.Name)
		if err := controllerutil.SetControllerReference(q, &ing, r.scheme); err != nil {
			return err
		}
		if err := r.client.Update(context.TODO(), &ing); err != nil {
			return err
		}
	}
	return nil
}

func (r *ReconcileQliksense) updateConfigMapOwner(q *qlikv1.Qliksense) error {

	listObj := &corev1.ConfigMapList{}
	if err := r.client.List(context.TODO(), listObj, client.MatchingLabels{searchingLabel: q.Name}); err != nil {
		return err
	}
	for _, cm := range listObj.Items {
		//fmt.Println(svc.Name)
		if err := controllerutil.SetControllerReference(q, &cm, r.scheme); err != nil {
			return err
		}
		if err := r.client.Update(context.TODO(), &cm); err != nil {
			return err
		}
	}
	return nil
}

func (r *ReconcileQliksense) updateSecretsOwner(q *qlikv1.Qliksense) error {

	listObj := &corev1.SecretList{}
	if err := r.client.List(context.TODO(), listObj, client.MatchingLabels{searchingLabel: q.Name}); err != nil {
		return err
	}
	for _, cm := range listObj.Items {
		//fmt.Println(svc.Name)
		if err := controllerutil.SetControllerReference(q, &cm, r.scheme); err != nil {
			return err
		}
		if err := r.client.Update(context.TODO(), &cm); err != nil {
			return err
		}
	}
	return nil
}

func (r *ReconcileQliksense) updatePvcOwner(q *qlikv1.Qliksense) error {

	listObj := &corev1.PersistentVolumeClaimList{}
	if err := r.client.List(context.TODO(), listObj, client.MatchingLabels{searchingLabel: q.Name}); err != nil {
		return err
	}
	for _, cm := range listObj.Items {

		if err := controllerutil.SetControllerReference(q, &cm, r.scheme); err != nil {
			return err
		}
		if err := r.client.Update(context.TODO(), &cm); err != nil {
			return err
		}
	}
	return nil
}

func (r *ReconcileQliksense) updateCronJobOwner(q *qlikv1.Qliksense) error {

	listObj := &batch_v1beta1.CronJobList{}
	if err := r.client.List(context.TODO(), listObj, client.MatchingLabels{searchingLabel: q.Name}); err != nil {
		return err
	}
	for _, cm := range listObj.Items {
		//fmt.Println(svc.Name)
		if err := controllerutil.SetControllerReference(q, &cm, r.scheme); err != nil {
			return err
		}
		if err := r.client.Update(context.TODO(), &cm); err != nil {
			return err
		}
	}
	return nil
}

func (r *ReconcileQliksense) updateServiceAccountOwner(q *qlikv1.Qliksense) error {

	listObj := &corev1.ServiceAccountList{}
	if err := r.client.List(context.TODO(), listObj, client.MatchingLabels{searchingLabel: q.Name}); err != nil {
		return err
	}
	for _, cm := range listObj.Items {
		//fmt.Println(svc.Name)
		if err := controllerutil.SetControllerReference(q, &cm, r.scheme); err != nil {
			return err
		}
		if err := r.client.Update(context.TODO(), &cm); err != nil {
			return err
		}
	}
	return nil
}

func (r *ReconcileQliksense) updateRoleOwner(q *qlikv1.Qliksense) error {

	listObj := &rbacv1.RoleList{}
	if err := r.client.List(context.TODO(), listObj, client.MatchingLabels{searchingLabel: q.Name}); err != nil {
		return err
	}
	for _, cm := range listObj.Items {
		//fmt.Println(svc.Name)
		if err := controllerutil.SetControllerReference(q, &cm, r.scheme); err != nil {
			return err
		}
		if err := r.client.Update(context.TODO(), &cm); err != nil {
			return err
		}
	}
	return nil
}

func (r *ReconcileQliksense) updateRoleBindingOwner(q *qlikv1.Qliksense) error {

	listObj := &rbacv1.RoleBindingList{}
	if err := r.client.List(context.TODO(), listObj, client.MatchingLabels{searchingLabel: q.Name}); err != nil {
		return err
	}
	for _, cm := range listObj.Items {
		//fmt.Println(svc.Name)
		if err := controllerutil.SetControllerReference(q, &cm, r.scheme); err != nil {
			return err
		}
		if err := r.client.Update(context.TODO(), &cm); err != nil {
			return err
		}
	}
	return nil
}

func (r *ReconcileQliksense) updateNetworkPolicyOwner(q *qlikv1.Qliksense) error {

	listObj := &v1beta1.NetworkPolicyList{}
	if err := r.client.List(context.TODO(), listObj, client.MatchingLabels{searchingLabel: q.Name}); err != nil {
		return err
	}
	for _, cm := range listObj.Items {
		//fmt.Println(svc.Name)
		if err := controllerutil.SetControllerReference(q, &cm, r.scheme); err != nil {
			return err
		}
		if err := r.client.Update(context.TODO(), &cm); err != nil {
			return err
		}
	}
	return nil
}

func (r *ReconcileQliksense) updateEngineOwner(q *qlikv1.Qliksense) error {
	// Get a config to talk to the apiserver
	cfg, err := config.GetConfig()
	if err != nil {
		return err
	}

	dynamicClient, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return err
	}

	engineRes := schema.GroupVersionResource{Group: "qixmanager.qlik.com", Version: "v1", Resource: "engines"}

	list, err := dynamicClient.Resource(engineRes).Namespace(q.Namespace).List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	// create owner reference object
	ref := *metav1.NewControllerRef(q, q.GroupVersionKind())

	for _, d := range list.Items {
		rls, _, err := unstructured.NestedString(d.Object, "metadata", "labels", searchingLabel)
		if err != nil {
			return err
		}
		if rls == q.Name {
			d.SetOwnerReferences([]metav1.OwnerReference{ref})
			if _, updateErr := dynamicClient.Resource(engineRes).Namespace(q.Namespace).Update(&d, metav1.UpdateOptions{}); updateErr != nil {
				return err
			}
		}
	}
	return nil
}
