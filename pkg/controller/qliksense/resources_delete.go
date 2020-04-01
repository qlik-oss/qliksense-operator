package qliksense

import (
	"context"

	"github.com/go-logr/logr"
	qlikv1 "github.com/qlik-oss/qliksense-operator/pkg/apis/qlik/v1"
	appsv1 "k8s.io/api/apps/v1"
	batch_v1 "k8s.io/api/batch/v1"
	batch_v1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

func (r *ReconcileQliksense) deleteDeployments(reqLogger logr.Logger, q *qlikv1.Qliksense) error {
	opts := []client.DeleteAllOfOption{
		client.InNamespace(q.GetNamespace()),
		client.MatchingLabels{searchingLabel: q.GetName()},
		client.GracePeriodSeconds(1),
	}
	if err := r.client.DeleteAllOf(context.TODO(), &appsv1.Deployment{}, opts...); err != nil {
		reqLogger.Error(err, "Cannot delete deployments")
		return nil
	}
	reqLogger.Info("Deleting Deployements")
	r.setCrStatus(reqLogger, q, "Valid", "DeletingDeployment", "User Initaited Action")
	return nil
}

func (r *ReconcileQliksense) deleteStatefuleSet(reqLogger logr.Logger, q *qlikv1.Qliksense) error {
	opts := []client.DeleteAllOfOption{
		client.InNamespace(q.GetNamespace()),
		client.MatchingLabels{searchingLabel: q.GetName()},
		client.GracePeriodSeconds(1),
	}
	if err := r.client.DeleteAllOf(context.TODO(), &appsv1.StatefulSet{}, opts...); err != nil {
		reqLogger.Error(err, "Cannot delete statefulset")
		return err
	}
	reqLogger.Info("Deleting Statefulset")
	r.setCrStatus(reqLogger, q, "Valid", "DeletingStatefulSet", "User Initaited Action")
	return nil
}

func (r *ReconcileQliksense) deleteCronJob(reqLogger logr.Logger, q *qlikv1.Qliksense) error {
	opts := []client.DeleteAllOfOption{
		client.InNamespace(q.GetNamespace()),
		client.MatchingLabels{searchingLabel: q.GetName()},
		client.GracePeriodSeconds(1),
	}
	if err := r.client.DeleteAllOf(context.TODO(), &batch_v1beta1.CronJob{}, opts...); err != nil {
		reqLogger.Error(err, "Cannot delete cronjob")
		return err
	}
	reqLogger.Info("Deleting CronJobs")
	r.setCrStatus(reqLogger, q, "Valid", "DeletingCronJob", "User Initaited Action")
	return nil
}

func (r *ReconcileQliksense) deleteJob(reqLogger logr.Logger, q *qlikv1.Qliksense) error {
	opts := []client.DeleteAllOfOption{
		client.InNamespace(q.GetNamespace()),
		client.MatchingLabels{searchingLabel: q.GetName()},
		client.GracePeriodSeconds(1),
	}
	if err := r.client.DeleteAllOf(context.TODO(), &batch_v1.Job{}, opts...); err != nil {
		reqLogger.Error(err, "Cannot delete job")
		return err
	}
	reqLogger.Info("Deleting Jobs")
	r.setCrStatus(reqLogger, q, "Valid", "DeletingJob", "User Initaited Action")
	return nil
}

func (r *ReconcileQliksense) deleteEngine(reqLogger logr.Logger, q *qlikv1.Qliksense) error {

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

	list, err := dynamicClient.Resource(engineRes).Namespace(q.Namespace).List(metav1.ListOptions{
		LabelSelector: searchingLabel + "=" + q.GetName(),
	})
	if err != nil {
		return err
	}
	var graceSec int64 = 1
	for _, d := range list.Items {
		if deleteErr := dynamicClient.Resource(engineRes).Namespace(q.Namespace).Delete(d.GetName(), &metav1.DeleteOptions{
			GracePeriodSeconds: &graceSec,
		}); deleteErr != nil {
			return err
		}
	}
	reqLogger.Info("Deleting Engines")
	r.setCrStatus(reqLogger, q, "Valid", "DeletingEngine", "User Initaited Action")
	return nil
}

func (r *ReconcileQliksense) deletePods(reqLogger logr.Logger, q *qlikv1.Qliksense) error {
	opts := []client.DeleteAllOfOption{
		client.InNamespace(q.GetNamespace()),
		client.MatchingLabels{searchingLabel: q.GetName()},
		client.GracePeriodSeconds(1),
	}
	if err := r.client.DeleteAllOf(context.TODO(), &corev1.Pod{}, opts...); err != nil {
		reqLogger.Error(err, "Cannot delete pods")
		return nil
	}
	r.setCrStatus(reqLogger, q, "Valid", "DeletingPods", "User Initaited Action")
	return nil
}

func (r *ReconcileQliksense) isAllResourceDeleted(reqLogger logr.Logger, q *qlikv1.Qliksense) bool {
	return r.isAllPodsDeleted(reqLogger, q)
}

func (r *ReconcileQliksense) isAllDeploymentsDeleted(reqLogger logr.Logger, q *qlikv1.Qliksense) bool {
	listObj := &appsv1.DeploymentList{}
	if err := r.client.List(context.TODO(), listObj, client.MatchingLabels{searchingLabel: q.Name}); err != nil {
		reqLogger.Error(err, "cannot find the list of deployments ")
		return false
	}
	return len(listObj.Items) == 0
}

func (r *ReconcileQliksense) isAllPodsDeleted(reqLogger logr.Logger, q *qlikv1.Qliksense) bool {
	listObj := &corev1.PodList{}
	if err := r.client.List(context.TODO(), listObj, client.MatchingLabels{searchingLabel: q.Name}); err != nil {
		reqLogger.Error(err, "cannot find the list of pods ")
		return false
	}
	return len(listObj.Items) == 0
}
