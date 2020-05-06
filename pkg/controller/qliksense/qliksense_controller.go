package qliksense

import (
	"context"
	"strconv"
	"time"

	"github.com/go-logr/logr"
	operator_status "github.com/operator-framework/operator-sdk/pkg/status"
	qlikv1 "github.com/qlik-oss/qliksense-operator/pkg/apis/qlik/v1"
	_ "gopkg.in/yaml.v2"
	appsv1 "k8s.io/api/apps/v1"
	batch_v1 "k8s.io/api/batch/v1"
	batch_v1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	_ "k8s.io/client-go/discovery"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_qliksense")

const (
	qliksenseFinalizer     = "finalizer.qliksense.qlik.com"
	searchingLabel         = "release"
	opsRunnerJobNameSuffix = "-ops-runner"
	maxDeletionWaitSeconds = 90 // 1.5 minutes
	pullSecretName         = "artifactory-docker-secret"
)

type OpsRunnerJobKind string

const (
	OpsRunnerJobKindCronJob    OpsRunnerJobKind = "CronJob"
	OpsRunnerJobKindRegularJob OpsRunnerJobKind = "RegularJob"
	OpsRunnerJobKindNone       OpsRunnerJobKind = "None"
)

type OpsRunnerJob struct {
	Kind OpsRunnerJobKind
	Job  interface{}
}

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
	return &ReconcileQliksense{client: mgr.GetClient(), scheme: mgr.GetScheme(), qlikInstances: NewQIs()}
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
	err = c.Watch(&source.Kind{Type: &corev1.Service{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &qlikv1.Qliksense{},
	}, predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			// Ignore updates to CR status in which case metadata.Generation does not change
			return e.MetaOld.GetGeneration() != e.MetaNew.GetGeneration()
		},
	})
	if err != nil {
		return err
	}
	err = c.Watch(&source.Kind{Type: &corev1.Secret{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &qlikv1.Qliksense{},
	}, predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			// Ignore updates to CR status in which case metadata.Generation does not change
			return e.MetaOld.GetGeneration() != e.MetaNew.GetGeneration()
		},
	})
	if err != nil {
		return err
	}
	err = c.Watch(&source.Kind{Type: &corev1.ConfigMap{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &qlikv1.Qliksense{},
	}, predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			// Ignore updates to CR status in which case metadata.Generation does not change
			return e.MetaOld.GetGeneration() != e.MetaNew.GetGeneration()
		},
	})
	if err != nil {
		return err
	}
	err = c.Watch(&source.Kind{Type: &corev1.ServiceAccount{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &qlikv1.Qliksense{},
	}, predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			// Ignore updates to CR status in which case metadata.Generation does not change
			return e.MetaOld.GetGeneration() != e.MetaNew.GetGeneration()
		},
	})
	if err != nil {
		return err
	}
	err = c.Watch(&source.Kind{Type: &corev1.PersistentVolumeClaim{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &qlikv1.Qliksense{},
	}, predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			// Ignore updates to CR status in which case metadata.Generation does not change
			return e.MetaOld.GetGeneration() != e.MetaNew.GetGeneration()
		},
	})
	if err != nil {
		return err
	}
	err = c.Watch(&source.Kind{Type: &appsv1.Deployment{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &qlikv1.Qliksense{},
	}, predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			// Ignore updates to CR status in which case metadata.Generation does not change
			return e.MetaOld.GetGeneration() != e.MetaNew.GetGeneration()
		},
	})
	if err != nil {
		return err
	}
	err = c.Watch(&source.Kind{Type: &appsv1.StatefulSet{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &qlikv1.Qliksense{},
	}, predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			// Ignore updates to CR status in which case metadata.Generation does not change
			return e.MetaOld.GetGeneration() != e.MetaNew.GetGeneration()
		},
	})
	if err != nil {
		return err
	}
	err = c.Watch(&source.Kind{Type: &v1beta1.Ingress{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &qlikv1.Qliksense{},
	}, predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			// Ignore updates to CR status in which case metadata.Generation does not change
			return e.MetaOld.GetGeneration() != e.MetaNew.GetGeneration()
		},
	})
	if err != nil {
		return err
	}

	err = c.Watch(&source.Kind{Type: &batch_v1beta1.CronJob{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &qlikv1.Qliksense{},
	}, predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			// Ignore updates to CR status in which case metadata.Generation does not change
			return e.MetaOld.GetGeneration() != e.MetaNew.GetGeneration()
		},
	})
	if err != nil {
		return err
	}
	err = c.Watch(&source.Kind{Type: &rbacv1.Role{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &qlikv1.Qliksense{},
	}, predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			// Ignore updates to CR status in which case metadata.Generation does not change
			return e.MetaOld.GetGeneration() != e.MetaNew.GetGeneration()
		},
	})
	if err != nil {
		return err
	}
	err = c.Watch(&source.Kind{Type: &rbacv1.RoleBinding{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &qlikv1.Qliksense{},
	}, predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			// Ignore updates to CR status in which case metadata.Generation does not change
			return e.MetaOld.GetGeneration() != e.MetaNew.GetGeneration()
		},
	})
	if err != nil {
		return err
	}

	//cannot watch engine resources. because we dont know the type yet
	return nil
}

// blank assignment to verify that ReconcileQliksense implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileQliksense{}

// ReconcileQliksense reconciles a Qliksense object
type ReconcileQliksense struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client        client.Client
	scheme        *runtime.Scheme
	qlikInstances *QliksenseInstances
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

	if !instance.Status.Conditions.IsTrueFor("Initialized") {
		r.setCrStatus(reqLogger, instance, "Valid", "Initialized", "")
	}
	// Check if the qliksense instance is marked to be deleted, which is
	// indicated by the deletion timestamp being set.
	isQliksenseMarkedToBeDeleted := instance.GetDeletionTimestamp() != nil
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

	// keep this for debugging pupose
	/*	if b, err := yaml.Marshal(instance); err != nil {
			reqLogger.Error(err, "cannot marshal qliksense CR")
		} else {
			fmt.Println(string(b))
		}
	*/

	// if no git then it was a CLI deployed k8 resources
	if instance.Spec.Git != nil && instance.Spec.Git.Repository != "" {
		if err := r.qlikInstances.AddToQliksenseInstances(instance); err != nil {
			reqLogger.Error(err, "Cannot create qliksense object")
			return reconcile.Result{}, nil
		}

		reqLogger.Info("Checking if Qliksense is installed...")
		if !r.qlikInstances.IsInstalled(instance.GetName()) {
			reqLogger.Info("Qliksense is not installed, installing...")
			// new install
			if err := r.qlikInstances.installQliksene(instance); err != nil {
				reqLogger.Error(err, "Cannot create kubernetes resoruces for "+instance.GetName())
				return reconcile.Result{}, err
			}
		} else {
			reqLogger.Info("Qliksense is already installed...")
		}
	} else {
		r.setCrStatus(reqLogger, instance, "Valid", "CliMode", "")
	}

	if instance.Spec.OpsRunner != nil {
		// next time jwt keys will not be updated
		instance.Spec.RotateKeys = "no"
		if err := r.setupOpsRunnerJob(reqLogger, instance); err != nil {
			return reconcile.Result{}, err
		}
		r.setCrStatus(reqLogger, instance, "Valid", "OpsRunnerMode", "")
	} else {
		r.setCrStatus(reqLogger, instance, "Valid", "GitMode", "")
	}

	if err := r.updateResourceOwner(reqLogger, instance); err != nil {
		r.setCrStatus(reqLogger, instance, "Valid", "Error", err.Error())
		return reconcile.Result{}, err
	}

	//reqLogger.Info("owner reference has been updated")

	// Add finalizer for this CR
	if !contains(instance.GetFinalizers(), qliksenseFinalizer) {
		if err := r.addFinalizer(reqLogger, instance); err != nil {
			return reconcile.Result{}, err
		}
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileQliksense) finalizeQliksense(reqLogger logr.Logger, qlik *qlikv1.Qliksense) error {
	// TODO(user): Add the cleanup steps that the operator
	// needs to do before the CR can be deleted. Examples
	// of finalizers include performing backups and deleting
	// resources that are not owned by this CR, like a PVC.
	// if err := r.client.DeleteAllOf(context.TODO(), &corev1.Service{}, client.MatchingLabels{searchingLabel: qlik.Name}); err != nil {
	// 	reqLogger.Error(err, "Cannot delete service")
	// 	return nil
	// }

	if err := r.qlikInstances.RemoveFromQliksenseInstances(qlik.GetName()); err != nil {
		reqLogger.Error(err, "cannot remove "+qlik.GetName()+" from instances")
	} else {

	}
	name := qlik.GetName()

	if err := r.deleteDeployments(reqLogger, qlik); err != nil {
		reqLogger.Error(err, "cannot delete deployments. Finalizing anyway")
		return nil
	}

	if err := r.deleteStatefuleSet(reqLogger, qlik); err != nil {
		reqLogger.Error(err, "cannot delete statefuleset. Finalizing anyway")
		return nil
	}
	if err := r.deleteCronJob(reqLogger, qlik); err != nil {
		reqLogger.Error(err, "cannot delete CronJob. Finalizing anyway")
		return nil
	}
	if err := r.deleteJob(reqLogger, qlik); err != nil {
		reqLogger.Error(err, "cannot delete Job. Finalizing anyway")
		return nil
	}
	if err := r.deleteEngine(reqLogger, qlik); err != nil {
		reqLogger.Error(err, "cannot delete Engine. Finalizing anyway")
		return nil
	}

	if err := r.deletePods(reqLogger, qlik); err != nil {
		reqLogger.Error(err, "cannot delete pods. Finalizing anyway")
		return nil
	}

	waitTimeCounter := 0
	for {
		time.Sleep(1 * time.Second)
		waitTimeCounter += 1
		reqLogger.Info("Waiting to finish resource deletion: " + strconv.Itoa(waitTimeCounter) + " seconds")
		if r.isAllPodsDeleted(reqLogger, qlik) || waitTimeCounter == maxDeletionWaitSeconds {
			break
		}
	}
	reqLogger.Info("Successfully finalized " + name)
	return nil
}

func (r *ReconcileQliksense) addFinalizer(reqLogger logr.Logger, m *qlikv1.Qliksense) error {
	reqLogger.Info("Adding Finalizer for the " + m.GetName())
	m.SetFinalizers(append(m.GetFinalizers(), qliksenseFinalizer))

	// Update CR
	err := r.client.Update(context.TODO(), m)
	if err != nil {
		reqLogger.Error(err, "Failed to update qliksense with finalizer")
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

func getRequiredOpsRunnerJobKind(m *qlikv1.Qliksense) OpsRunnerJobKind {
	if m.Spec.OpsRunner.Enabled == "yes" {
		if m.Spec.OpsRunner.Schedule != "" {
			return OpsRunnerJobKindCronJob
		}
		return OpsRunnerJobKindRegularJob
	}
	return OpsRunnerJobKindNone
}

func (r *ReconcileQliksense) getCurrentOpsRunnerJob(reqLogger logr.Logger, m *qlikv1.Qliksense) (*OpsRunnerJob, error) {
	reqLogger.Info("Trying to fetch OpsRunner CronJob...")
	cronJob := &batch_v1beta1.CronJob{}
	if err := r.client.Get(context.TODO(), types.NamespacedName{Name: m.Name + opsRunnerJobNameSuffix, Namespace: m.Namespace}, cronJob); err == nil {
		return &OpsRunnerJob{
			Kind: OpsRunnerJobKindCronJob,
			Job:  cronJob,
		}, nil
	} else if !errors.IsNotFound(err) {
		return nil, err
	}

	reqLogger.Info("Trying to fetch OpsRunner regular Job...")
	regularJob := &batch_v1.Job{}
	if err := r.client.Get(context.TODO(), types.NamespacedName{Name: m.Name + opsRunnerJobNameSuffix, Namespace: m.Namespace}, regularJob); err == nil {
		return &OpsRunnerJob{
			Kind: OpsRunnerJobKindRegularJob,
			Job:  regularJob,
		}, nil
	} else if !errors.IsNotFound(err) {
		return nil, err
	}

	reqLogger.Info("Neither the OpsRunner CronJob nor the regular Job were found...")
	return &OpsRunnerJob{
		Kind: OpsRunnerJobKindNone,
		Job:  nil,
	}, nil
}

func (r *ReconcileQliksense) deleteCurrentOpsRunnerJob(reqLogger logr.Logger, opsRunnerJob *OpsRunnerJob) error {
	if opsRunnerJob.Kind == OpsRunnerJobKindCronJob {
		reqLogger.Info("Deleting OpsRunner CronJob")
		return r.client.Delete(context.TODO(), opsRunnerJob.Job.(*batch_v1beta1.CronJob))
	} else if opsRunnerJob.Kind == OpsRunnerJobKindRegularJob {
		reqLogger.Info("Deleting OpsRunner Job")
		return r.client.Delete(context.TODO(), opsRunnerJob.Job.(*batch_v1.Job))
	}
	reqLogger.Info("Nothing to delete")
	return nil
}

func (r *ReconcileQliksense) createOpsRunnerJob(opsRunnerJobKind OpsRunnerJobKind, reqLogger logr.Logger, m *qlikv1.Qliksense) error {
	if opsRunnerJobKind == OpsRunnerJobKindCronJob {
		reqLogger.Info("Configuring the OpsRunner CronJob...")
		if contJob, err := r.getOpsRunnerCronJob(reqLogger, m); err != nil {
			return err
		} else {
			reqLogger.Info("Creating the OpsRunner CronJob", "CronJob.Namespace", contJob.Namespace, "CronJob.Name", contJob.Name)
			return r.createAndValidateK8sJobObject(reqLogger, contJob, &contJob.ObjectMeta)
		}
	} else if opsRunnerJobKind == OpsRunnerJobKindRegularJob {
		reqLogger.Info("Configuring the OpsRunner regular Job...")
		if job, err := r.getOpsRunnerJob(reqLogger, m); err != nil {
			return err
		} else {
			reqLogger.Info("Creating the OpsRunner job", "CronJob.Namespace", job.Namespace, "CronJob.Name", job.Name)
			return r.createAndValidateK8sJobObject(reqLogger, job, &job.ObjectMeta)
		}
	}
	reqLogger.Info("Nothing to create")
	return nil
}

func (r *ReconcileQliksense) createAndValidateK8sJobObject(reqLogger logr.Logger, job runtime.Object, jobMetadata *metav1.ObjectMeta) error {
	if err := r.client.Create(context.TODO(), job); err == nil {
		reqLogger.Info("Successfully created the OpsRunner job", "namespace", jobMetadata.Namespace, "name", jobMetadata.Name)
		return nil
	} else if errors.IsAlreadyExists(err) {
		reqLogger.Info("OpsRunner job already exists", "namespace", jobMetadata.Namespace, "name", jobMetadata.Name)
		return nil
	} else {
		reqLogger.Error(err, "Failed to create the OpsRunner job", "namespace", jobMetadata.Namespace, "name", jobMetadata.Name)
		return err
	}
}

// setupOpsRunnerJob create a new job if it did not exist before, and delete an existing job if enabled=no
func (r *ReconcileQliksense) setupOpsRunnerJob(reqLogger logr.Logger, m *qlikv1.Qliksense) error {
	requiredOpsRunnerJobKind := getRequiredOpsRunnerJobKind(m)
	currentOpsRunnerJob, err := r.getCurrentOpsRunnerJob(reqLogger, m)
	if err != nil {
		reqLogger.Error(err, "Failed to retrieve current OpsRunner job")
		return err
	}
	if requiredOpsRunnerJobKind == OpsRunnerJobKindNone {
		if err := r.deleteCurrentOpsRunnerJob(reqLogger, currentOpsRunnerJob); err != nil {
			reqLogger.Error(err, "Failed to delete current OpsRunner job")
			return err
		}
	} else {
		if err := r.createOpsRunnerJob(requiredOpsRunnerJobKind, reqLogger, m); err != nil {
			reqLogger.Error(err, "Failed to delete current OpsRunner job")
			return err
		}
	}
	return nil
}

func (r *ReconcileQliksense) setCrStatus(reqLogger logr.Logger, m *qlikv1.Qliksense, sts, tps, reason string) error {
	var cond operator_status.Condition
	cond = operator_status.Condition{
		Type:   operator_status.ConditionType(tps),
		Status: corev1.ConditionStatus(sts),
	}

	if reason != "" {
		cond.Reason = operator_status.ConditionReason(reason)
	}

	m.Status.Conditions.SetCondition(cond)
	return r.client.Status().Update(context.TODO(), m)
}
