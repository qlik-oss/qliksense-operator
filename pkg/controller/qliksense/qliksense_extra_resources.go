package qliksense

import (
	"fmt"
	"path"
	"strings"

	"github.com/go-logr/logr"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	qlikv1 "github.com/qlik-oss/qliksense-operator/pkg/apis/qlik/v1"
	batch_v1 "k8s.io/api/batch/v1"
	batch_v1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *ReconcileQliksense) getOpsRunnerCronJob(reqLogger logr.Logger, m *qlikv1.Qliksense) (*batch_v1beta1.CronJob, error) {
	podSpec, err := r.getJobPodSpec(reqLogger, m)
	if err != nil {
		return nil, err
	}
	cronJob := &batch_v1beta1.CronJob{
		ObjectMeta: *getJobMetadata(m),
		Spec: batch_v1beta1.CronJobSpec{
			Schedule: m.Spec.OpsRunner.Schedule,
			JobTemplate: batch_v1beta1.JobTemplateSpec{
				Spec: batch_v1.JobSpec{
					Template: corev1.PodTemplateSpec{
						Spec: *podSpec,
					},
				},
			},
		},
	}

	if err := controllerutil.SetControllerReference(m, cronJob, r.scheme); err != nil {
		reqLogger.Error(err, "Error setting controller reference for cronJob")
		return nil, err
	}
	return cronJob, nil
}

func (r *ReconcileQliksense) getOpsRunnerJob(reqLogger logr.Logger, m *qlikv1.Qliksense) (*batch_v1.Job, error) {
	podSpec, err := r.getJobPodSpec(reqLogger, m)
	if err != nil {
		reqLogger.Error(err, "Error configuring the job PodSpec")
		return nil, err
	}
	job := &batch_v1.Job{
		ObjectMeta: *getJobMetadata(m),
		Spec: batch_v1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: *podSpec,
			},
		},
	}

	if err := controllerutil.SetControllerReference(m, job, r.scheme); err != nil {
		reqLogger.Error(err, "Error setting controller reference for job")
		return nil, err
	}
	return job, nil
}

func getJobMetadata(m *qlikv1.Qliksense) *metav1.ObjectMeta {
	return &metav1.ObjectMeta{
		Name:      m.Name + opsRunnerJobNameSuffix,
		Namespace: m.Namespace,
		Labels: map[string]string{
			"release": m.Name,
		},
	}
}

func (r *ReconcileQliksense) getJobPodSpec(reqLogger logr.Logger, m *qlikv1.Qliksense) (*corev1.PodSpec, error) {
	operatorName, err := k8sutil.GetOperatorName()
	if err != nil {
		reqLogger.Error(err, "Error obtaining operator name")
		return nil, err
	}
	b, err := K8sToYaml(m)
	if err != nil {
		reqLogger.Error(err, "Error marshalling CR to yaml")
		return nil, err
	}
	podSpec := &corev1.PodSpec{
		Containers: []corev1.Container{{
			Image:           m.Spec.OpsRunner.Image,
			ImagePullPolicy: corev1.PullIfNotPresent,
			Name:            m.Name + opsRunnerJobNameSuffix,
			Env: []corev1.EnvVar{
				{
					Name:  "YAML_CONF",
					Value: string(b),
				},
				{
					Name:  "OPERATOR_SERVICE_NAME",
					Value: fmt.Sprintf("%s-kuztomize", operatorName),
				},
				{
					Name:  "OPERATOR_SERVICE_PORT",
					Value: fmt.Sprintf("%v", kuzServicePort),
				},
			},
		}},
		RestartPolicy: corev1.RestartPolicyNever,
	}
	updateJobPodSpecForImageRegistry(m, podSpec)
	return podSpec, nil
}

func updateJobPodSpecForImageRegistry(m *qlikv1.Qliksense, podTemplateSpec *corev1.PodSpec) {
	if imageRegistry := m.Spec.GetImageRegistry(); imageRegistry != "" {
		if currentImage := podTemplateSpec.Containers[0].Image; currentImage != "" {
			imageSegments := strings.Split(currentImage, "/")
			imageNameAndTag := imageSegments[len(imageSegments)-1]
			podTemplateSpec.Containers[0].Image = path.Join(imageRegistry, imageNameAndTag)
			podTemplateSpec.ImagePullSecrets = []corev1.LocalObjectReference{{Name: pullSecretName}}
		}
	}
}
