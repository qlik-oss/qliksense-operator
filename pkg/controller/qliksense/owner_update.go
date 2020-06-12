package qliksense

import (
	"context"

	"k8s.io/apimachinery/pkg/api/errors"

	"github.com/go-logr/logr"
	qlikv1 "github.com/qlik-oss/qliksense-operator/pkg/apis/qlik/v1"
	appsv1 "k8s.io/api/apps/v1"
	batch_v1 "k8s.io/api/batch/v1"
	batch_v1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	networking_v1 "k8s.io/api/networking/v1"
	networking_v1beta1 "k8s.io/api/networking/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *ReconcileQliksense) updateResourceOwner(reqLogger logr.Logger, instance *qlikv1.Qliksense) error {
	if err := r.updateServiceOwner(reqLogger, instance); err != nil {
		reqLogger.Error(err, "cannot update service owner")
		return err
	}
	if err := r.updateDeploymentOwner(reqLogger, instance); err != nil {
		reqLogger.Error(err, "cannot update deployments owner")
		return err
	}
	if err := r.updateStatefulSetOwner(reqLogger, instance); err != nil {
		reqLogger.Error(err, "cannot update deployments owner")
		return err
	}
	if err := r.updateConfigMapOwner(reqLogger, instance); err != nil {
		reqLogger.Error(err, "cannot update config map owner")
		return err
	}
	if err := r.updateSecretsOwner(reqLogger, instance); err != nil {
		reqLogger.Error(err, "cannot update secrets owner")
		return err
	}
	if err := r.updatePvcOwner(reqLogger, instance); err != nil {
		reqLogger.Error(err, "cannot update pvc owner")
		return err
	}
	if err := r.updateCronJobOwner(reqLogger, instance); err != nil {
		reqLogger.Error(err, "cannot update cronjob owner")
		return err
	}
	if err := r.updateJobOwner(reqLogger, instance); err != nil {
		reqLogger.Error(err, "cannot update job owner")
		return err
	}
	if err := r.updateServiceAccountOwner(reqLogger, instance); err != nil {
		reqLogger.Error(err, "cannot update service account owner")
		return err
	}
	if err := r.updateRoleOwner(reqLogger, instance); err != nil {
		reqLogger.Error(err, "cannot update role owner")
		return err
	}
	if err := r.updateRoleBindingOwner(reqLogger, instance); err != nil {
		reqLogger.Error(err, "cannot update role binding owner")
		return err
	}
	if err := r.updateNetworkPolicyOwner(reqLogger, instance); err != nil {
		reqLogger.Error(err, "cannot update network policy owner")
		return err
	}
	if err := r.updateIngressOwner(reqLogger, instance); err != nil {
		reqLogger.Error(err, "cannot update ingress owner")
		return err
	}

	customResources := []schema.GroupVersionResource{
		{Group: "qixmanager.qlik.com", Version: "v1", Resource: "engines"},
		{Group: "qixengine.qlik.com", Version: "v1", Resource: "engines"},
		{Group: "qixengine.qlik.com", Version: "v1", Resource: "enginetemplates"},
		{Group: "qixengine.qlik.com", Version: "v1", Resource: "enginevariants"},
	}
	for _, customResource := range customResources {
		if err := r.updateGroupVersionResourceOwner(reqLogger, instance, customResource); err != nil {
			reqLogger.Error(err, "cannot update custom resource owner using dynamic client", "GroupVersionResource", customResource)
			return err
		}
	}

	regularResources := []schema.GroupVersionResource{
		{Group: "autoscaling", Version: "v1", Resource: "horizontalpodautoscalers"},
	}
	for _, regularResource := range regularResources {
		if err := r.updateGroupVersionResourceOwner(reqLogger, instance, regularResource); err != nil {
			reqLogger.Error(err, "cannot update regular resource owner using dynamic client", "GroupVersionResource", regularResource)
			return err
		}
	}

	return nil
}

func (r *ReconcileQliksense) updateServiceOwner(reqLogger logr.Logger, q *qlikv1.Qliksense) error {

	listObj := &corev1.ServiceList{}
	if err := r.client.List(context.TODO(), listObj, client.MatchingLabels{searchingLabel: q.Name}); err != nil {
		return err
	}
	for _, svc := range listObj.Items {
		alreadySet := false
		for _, or := range svc.GetOwnerReferences() {
			if or.Name == q.GetName() {
				alreadySet = true
				break
			}
		}
		if alreadySet {
			continue
		}
		if err := controllerutil.SetControllerReference(q, &svc, r.scheme); err != nil {
			return err
		} else if err := r.client.Update(context.TODO(), &svc); err != nil {
			return err
		}
		reqLogger.Info("update owner for service [ " + svc.Name + " ]")
	}
	return nil
}

func (r *ReconcileQliksense) updateDeploymentOwner(reqLogger logr.Logger, q *qlikv1.Qliksense) error {

	listObj := &appsv1.DeploymentList{}
	if err := r.client.List(context.TODO(), listObj, client.MatchingLabels{searchingLabel: q.Name}); err != nil {
		return err
	}
	for _, dep := range listObj.Items {
		alreadySet := false
		for _, or := range dep.GetOwnerReferences() {
			if or.Name == q.GetName() {
				alreadySet = true
				break
			}
		}
		if alreadySet {
			continue
		}
		if err := controllerutil.SetControllerReference(q, &dep, r.scheme); err != nil {
			return err
		} else if err := r.client.Update(context.TODO(), &dep); err != nil {
			return err
		}
		reqLogger.Info("update owner for deployment [ " + dep.Name + " ]")
	}
	return nil
}

func (r *ReconcileQliksense) updateStatefulSetOwner(reqLogger logr.Logger, q *qlikv1.Qliksense) error {

	listObj := &appsv1.StatefulSetList{}
	if err := r.client.List(context.TODO(), listObj, client.MatchingLabels{searchingLabel: q.Name}); err != nil {
		return err
	}
	for _, dep := range listObj.Items {
		alreadySet := false
		for _, or := range dep.GetOwnerReferences() {
			if or.Name == q.GetName() {
				alreadySet = true
				break
			}
		}
		if alreadySet {
			continue
		}
		if err := controllerutil.SetControllerReference(q, &dep, r.scheme); err != nil {
			return err
		} else if err := r.client.Update(context.TODO(), &dep); err != nil {
			return err
		}
		reqLogger.Info("update owner for statefulset [ " + dep.Name + " ]")
	}
	return nil
}

func (r *ReconcileQliksense) updateIngressOwner(reqLogger logr.Logger, q *qlikv1.Qliksense) error {

	listObj := &networking_v1beta1.IngressList{}
	if err := r.client.List(context.TODO(), listObj, client.MatchingLabels{searchingLabel: q.Name}); err != nil {
		return err
	}
	for _, ing := range listObj.Items {
		alreadySet := false
		for _, or := range ing.GetOwnerReferences() {
			if or.Name == q.GetName() {
				alreadySet = true
				break
			}
		}
		if alreadySet {
			continue
		}
		if err := controllerutil.SetControllerReference(q, &ing, r.scheme); err != nil {
			return err
		} else if err := r.client.Update(context.TODO(), &ing); err != nil {
			return err
		}
		reqLogger.Info("update owner for Ingress [ " + ing.Name + " ]")
	}
	return nil
}

func (r *ReconcileQliksense) updateConfigMapOwner(reqLogger logr.Logger, q *qlikv1.Qliksense) error {

	listObj := &corev1.ConfigMapList{}
	if err := r.client.List(context.TODO(), listObj, client.MatchingLabels{searchingLabel: q.Name}); err != nil {
		return err
	}
	for _, cm := range listObj.Items {
		alreadySet := false
		for _, or := range cm.GetOwnerReferences() {
			if or.Name == q.GetName() {
				alreadySet = true
				break
			}
		}
		if alreadySet {
			continue
		}
		if err := controllerutil.SetControllerReference(q, &cm, r.scheme); err != nil {
			return err
		} else if err := r.client.Update(context.TODO(), &cm); err != nil {
			return err
		}
		reqLogger.Info("update owner for ConfigMap [ " + cm.Name + " ]")
	}
	return nil
}

func (r *ReconcileQliksense) updateSecretsOwner(reqLogger logr.Logger, q *qlikv1.Qliksense) error {

	listObj := &corev1.SecretList{}
	if err := r.client.List(context.TODO(), listObj, client.MatchingLabels{searchingLabel: q.Name}); err != nil {
		return err
	}
	for _, cm := range listObj.Items {
		alreadySet := false
		for _, or := range cm.GetOwnerReferences() {
			if or.Name == q.GetName() {
				alreadySet = true
				break
			}
		}
		if alreadySet {
			continue
		}
		if err := controllerutil.SetControllerReference(q, &cm, r.scheme); err != nil {
			return err
		} else if err := r.client.Update(context.TODO(), &cm); err != nil {
			return err
		}
		reqLogger.Info("update owner for Secrets [ " + cm.Name + " ]")
	}
	return nil
}

func (r *ReconcileQliksense) updatePvcOwner(reqLogger logr.Logger, q *qlikv1.Qliksense) error {

	listObj := &corev1.PersistentVolumeClaimList{}
	if err := r.client.List(context.TODO(), listObj, client.MatchingLabels{searchingLabel: q.Name}); err != nil {
		return err
	}
	for _, cm := range listObj.Items {
		alreadySet := false
		for _, or := range cm.GetOwnerReferences() {
			if or.Name == q.GetName() {
				alreadySet = true
				break
			}
		}
		if alreadySet {
			continue
		}
		if err := controllerutil.SetControllerReference(q, &cm, r.scheme); err != nil {
			return err
		} else if err := r.client.Update(context.TODO(), &cm); err != nil {
			return err
		}
		reqLogger.Info("update owner for pvc [ " + cm.Name + " ]")
	}
	return nil
}

func (r *ReconcileQliksense) updateCronJobOwner(reqLogger logr.Logger, q *qlikv1.Qliksense) error {

	listObj := &batch_v1beta1.CronJobList{}
	if err := r.client.List(context.TODO(), listObj, client.MatchingLabels{searchingLabel: q.Name}); err != nil {
		return err
	}
	for _, cm := range listObj.Items {
		alreadySet := false
		for _, or := range cm.GetOwnerReferences() {
			if or.Name == q.GetName() {
				alreadySet = true
				break
			}
		}
		if alreadySet {
			continue
		}
		if err := controllerutil.SetControllerReference(q, &cm, r.scheme); err != nil {
			return err
		} else if err := r.client.Update(context.TODO(), &cm); err != nil {
			return err
		}
		reqLogger.Info("update owner for CronJob [ " + cm.Name + " ]")
	}
	return nil
}
func (r *ReconcileQliksense) updateJobOwner(reqLogger logr.Logger, q *qlikv1.Qliksense) error {

	listObj := &batch_v1.JobList{}
	if err := r.client.List(context.TODO(), listObj, client.MatchingLabels{searchingLabel: q.Name}); err != nil {
		return err
	}
	for _, job := range listObj.Items {
		alreadySet := false
		for _, or := range job.GetOwnerReferences() {
			if or.Name == q.GetName() {
				alreadySet = true
				break
			}
		}
		if alreadySet {
			continue
		}
		if err := controllerutil.SetControllerReference(q, &job, r.scheme); err != nil {
			if alreadyOwnedError, isAlreadyOwnedError := err.(*controllerutil.AlreadyOwnedError); !isAlreadyOwnedError || alreadyOwnedError.Owner.Kind != "CronJob" {
				return err
			}
		} else if err := r.client.Update(context.TODO(), &job); err != nil {
			return err
		} else {
			reqLogger.Info("update owner for Job [ " + job.Name + " ]")
		}
	}
	return nil
}

func (r *ReconcileQliksense) updateServiceAccountOwner(reqLogger logr.Logger, q *qlikv1.Qliksense) error {

	listObj := &corev1.ServiceAccountList{}
	if err := r.client.List(context.TODO(), listObj, client.MatchingLabels{searchingLabel: q.Name}); err != nil {
		return err
	}
	for _, cm := range listObj.Items {
		alreadySet := false
		for _, or := range cm.GetOwnerReferences() {
			if or.Name == q.GetName() {
				alreadySet = true
				break
			}
		}
		if alreadySet {
			continue
		}
		if err := controllerutil.SetControllerReference(q, &cm, r.scheme); err != nil {
			return err
		} else if err := r.client.Update(context.TODO(), &cm); err != nil {
			return err
		}
		reqLogger.Info("update owner for ServiceAccount [ " + cm.Name + " ]")
	}
	return nil
}

func (r *ReconcileQliksense) updateRoleOwner(reqLogger logr.Logger, q *qlikv1.Qliksense) error {

	listObj := &rbacv1.RoleList{}
	if err := r.client.List(context.TODO(), listObj, client.MatchingLabels{searchingLabel: q.Name}); err != nil {
		return err
	}
	for _, cm := range listObj.Items {
		alreadySet := false
		for _, or := range cm.GetOwnerReferences() {
			if or.Name == q.GetName() {
				alreadySet = true
				break
			}
		}
		if alreadySet {
			continue
		}
		if err := controllerutil.SetControllerReference(q, &cm, r.scheme); err != nil {
			return err
		} else if err := r.client.Update(context.TODO(), &cm); err != nil {
			return err
		}
		reqLogger.Info("update owner for Role [ " + cm.Name + " ]")
	}
	return nil
}

func (r *ReconcileQliksense) updateRoleBindingOwner(reqLogger logr.Logger, q *qlikv1.Qliksense) error {

	listObj := &rbacv1.RoleBindingList{}
	if err := r.client.List(context.TODO(), listObj, client.MatchingLabels{searchingLabel: q.Name}); err != nil {
		return err
	}
	for _, cm := range listObj.Items {
		alreadySet := false
		for _, or := range cm.GetOwnerReferences() {
			if or.Name == q.GetName() {
				alreadySet = true
				break
			}
		}
		if alreadySet {
			continue
		}
		if err := controllerutil.SetControllerReference(q, &cm, r.scheme); err != nil {
			return err
		} else if err := r.client.Update(context.TODO(), &cm); err != nil {
			return err
		}
		reqLogger.Info("update owner for RoleBinding [ " + cm.Name + " ]")
	}
	return nil
}

func (r *ReconcileQliksense) updateNetworkPolicyOwner(reqLogger logr.Logger, q *qlikv1.Qliksense) error {

	listObj := &networking_v1.NetworkPolicyList{}
	if err := r.client.List(context.TODO(), listObj, client.MatchingLabels{searchingLabel: q.Name}); err != nil {
		return err
	}
	for _, cm := range listObj.Items {
		alreadySet := false
		for _, or := range cm.GetOwnerReferences() {
			if or.Name == q.GetName() {
				alreadySet = true
				break
			}
		}
		if alreadySet {
			continue
		}
		if err := controllerutil.SetControllerReference(q, &cm, r.scheme); err != nil {
			return err
		} else if err := r.client.Update(context.TODO(), &cm); err != nil {
			return err
		}
		reqLogger.Info("update owner for NetworkPolicy [ " + cm.Name + " ]")
	}
	return nil
}

// TODO: use dynamic client for all other standard resources, so that only one method can be used
func (r *ReconcileQliksense) updateGroupVersionResourceOwner(reqLogger logr.Logger, q *qlikv1.Qliksense, groupVersionResource schema.GroupVersionResource) error {
	// Get a config to talk to the apiserver
	cfg, err := config.GetConfig()
	if err != nil {
		return err
	}

	dynamicClient, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return err
	}

	list, err := dynamicClient.Resource(groupVersionResource).Namespace(q.Namespace).List(metav1.ListOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			reqLogger.Info("WARNING: cannot update ownership because resources are not found", "GroupVersionResource", groupVersionResource)
			return nil
		}
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
			alreadySet := false
			for _, or := range d.GetOwnerReferences() {
				if or.Name == q.GetName() {
					alreadySet = true
					break
				}
			}
			if alreadySet {
				continue
			}
			d.SetOwnerReferences([]metav1.OwnerReference{ref})
			if _, updateErr := dynamicClient.Resource(groupVersionResource).Namespace(q.Namespace).Update(&d, metav1.UpdateOptions{}); updateErr != nil {
				return err
			}
		}
		reqLogger.Info("update owner for resource", "GroupVersionResource", groupVersionResource)
	}
	return nil
}
