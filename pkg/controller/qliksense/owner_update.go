package qliksense

import (
	"context"
	"github.com/go-logr/logr"
	qlikv1 "github.com/qlik-oss/qliksense-operator/pkg/apis/qlik/v1"
	appsv1 "k8s.io/api/apps/v1"
	batch_v1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
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
	if err := r.updateServiceOwner(instance); err != nil {
		reqLogger.Error(err, "cannot update service owner")
		return err
	}
	if err := r.updateDeploymentOwner(instance); err != nil {
		reqLogger.Error(err, "cannot update deployments owner")
		return err
	}
	if err := r.updateStatefulSetOwner(instance); err != nil {
		reqLogger.Error(err, "cannot update deployments owner")
		return err
	}
	if err := r.updateConfigMapOwner(instance); err != nil {
		reqLogger.Error(err, "cannot update config map owner")
		return err
	}
	if err := r.updateSecretsOwner(instance); err != nil {
		reqLogger.Error(err, "cannot update secrets owner")
		return err
	}
	if err := r.updatePvcOwner(instance); err != nil {
		reqLogger.Error(err, "cannot update pvc owner")
		return err
	}
	if err := r.updateCronJobOwner(instance); err != nil {
		reqLogger.Error(err, "cannot update cronjob owner")
		return err
	}
	if err := r.updateServiceAccountOwner(instance); err != nil {
		reqLogger.Error(err, "cannot update service account owner")
		return err
	}
	if err := r.updateRoleOwner(instance); err != nil {
		reqLogger.Error(err, "cannot update role owner")
		return err
	}
	if err := r.updateRoleBindingOwner(instance); err != nil {
		reqLogger.Error(err, "cannot update role binding owner")
		return err
	}
	if err := r.updateNetworkPolicyOwner(instance); err != nil {
		reqLogger.Error(err, "cannot update network policy owner")
		return err
	}
	if err := r.updateEngineOwner(instance); err != nil {
		reqLogger.Error(err, "cannot update Engine owner")
		return err
	}

	return nil
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
