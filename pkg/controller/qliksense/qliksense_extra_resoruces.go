package qliksense

import (
	"fmt"
	"path"
	"strings"

	"github.com/operator-framework/operator-sdk/pkg/k8sutil"

	"github.com/qlik-oss/qliksense-operator/cmd/server"

	"github.com/go-logr/logr"
	qlikv1 "github.com/qlik-oss/qliksense-operator/pkg/apis/qlik/v1"
	batch_v1 "k8s.io/api/batch/v1"
	batch_v1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// jobForExecutor returns a QseokExecutor Job object
func (r *ReconcileQliksense) cronJobForGitOps(reqLogger logr.Logger, m *qlikv1.Qliksense) (*batch_v1beta1.CronJob, error) {
	b, err := K8sToYaml(m)
	if err != nil {
		return nil, err
	}
	operatorName, err := k8sutil.GetOperatorName()
	if err != nil {
		return nil, err
	}
	cronJob := &batch_v1beta1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.Name + gitOpsCJSuffix,
			Namespace: m.Namespace,
			Labels: map[string]string{
				"release": m.Name,
			},
		},
		Spec: batch_v1beta1.CronJobSpec{
			Schedule: m.Spec.OpsRunner.Schedule,
			JobTemplate: batch_v1beta1.JobTemplateSpec{
				Spec: batch_v1.JobSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{{
								Image:           m.Spec.OpsRunner.Image,
								ImagePullPolicy: corev1.PullIfNotPresent,
								Name:            m.Name + gitOpsCJSuffix,
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
										Value: fmt.Sprintf("%v", server.KuzPort),
									},
								},
							}},
							RestartPolicy: "OnFailure",
							//Volumes: []corev1.Volume{{
							//	Name: m.Name + gitOpsCJSuffix,
							//	VolumeSource: corev1.VolumeSource{
							//		PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
							//			ClaimName: m.Name + gitOpsCJSuffix,
							//		},
							//	},
							//}},
						},
					},
				},
			},
		},
	}

	updateCronJobForImageRegistry(m, cronJob)
	controllerutil.SetControllerReference(m, cronJob, r.scheme)
	return cronJob, nil
}

func updateCronJobForImageRegistry(m *qlikv1.Qliksense, cronJob *batch_v1beta1.CronJob) {
	if imageRegistry := m.Spec.GetImageRegistry(); imageRegistry != "" {
		podTemplateSpec := &cronJob.Spec.JobTemplate.Spec.Template.Spec
		if currentImage := podTemplateSpec.Containers[0].Image; currentImage != "" {
			imageSegments := strings.Split(currentImage, "/")
			imageNameAndTag := imageSegments[len(imageSegments)-1]
			podTemplateSpec.Containers[0].Image = path.Join(imageRegistry, imageNameAndTag)
			podTemplateSpec.ImagePullSecrets = []corev1.LocalObjectReference{{Name: pullSecretName}}
		}
	}
}

// jobForRunner returns a runner Job object
func (r *ReconcileQliksense) jobForRunner(reqLogger logr.Logger, m *qlikv1.Qliksense) (*batch_v1.Job, error) {
	b, err := K8sToYaml(m)
	if err != nil {
		return nil, err
	}

	job := &batch_v1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.Name + gitOpsCJSuffix,
			Namespace: m.Namespace,
			Labels: map[string]string{
				"release": m.Name,
			},
		},
		Spec: batch_v1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image: m.Spec.OpsRunner.Image,
						Name:  m.Name + gitOpsCJSuffix,
						Env: []corev1.EnvVar{
							{
								Name:  "YAML_CONF",
								Value: string(b),
							},
						},
					}},
					RestartPolicy: "OnFailure",
					Volumes: []corev1.Volume{{
						Name: m.Name + gitOpsCJSuffix,
						VolumeSource: corev1.VolumeSource{
							PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
								ClaimName: m.Name + gitOpsCJSuffix,
							},
						},
					}},
				},
			},
		},
	}

	updateJobForImageRegistry(m, job)
	controllerutil.SetControllerReference(m, job, r.scheme)
	return job, nil
}

func updateJobForImageRegistry(m *qlikv1.Qliksense, job *batch_v1.Job) {
	if imageRegistry := m.Spec.GetImageRegistry(); imageRegistry != "" {
		podTemplateSpec := &job.Spec.Template.Spec
		if currentImage := podTemplateSpec.Containers[0].Image; currentImage != "" {
			imageSegments := strings.Split(currentImage, "/")
			imageNameAndTag := imageSegments[len(imageSegments)-1]
			podTemplateSpec.Containers[0].Image = path.Join(imageRegistry, imageNameAndTag)
			podTemplateSpec.ImagePullSecrets = []corev1.LocalObjectReference{{Name: pullSecretName}}
		}
	}
}

// pvcForRunner returns a persitentvolume claim object
func (r *ReconcileQliksense) pvcForRunner(reqLogger logr.Logger, m *qlikv1.Qliksense) (*corev1.PersistentVolumeClaim, error) {
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.Name + gitOpsCJSuffix,
			Namespace: m.Namespace,
			Labels: map[string]string{
				"release": m.Name,
			},
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse("1Gi"),
				},
			},
		},
	}
	if m.Spec.StorageClassName != "" {
		pvc.Spec.StorageClassName = &m.Spec.StorageClassName
	}
	controllerutil.SetControllerReference(m, pvc, r.scheme)
	return pvc, nil
}
