package qliksense

import (
	"fmt"
	"os"
	"path"
	"strings"

	"sigs.k8s.io/yaml"

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
	podSpec := corev1.PodSpec{}
	if err := r.updateJobPodSpec(&podSpec, reqLogger, m); err != nil {
		return nil, err
	}

	objectMeta := metav1.ObjectMeta{}
	updateJobMetadata(&objectMeta, m)

	cronJob := &batch_v1beta1.CronJob{
		ObjectMeta: objectMeta,
		Spec: batch_v1beta1.CronJobSpec{
			Schedule:          m.Spec.OpsRunner.Schedule,
			ConcurrencyPolicy: batch_v1beta1.ForbidConcurrent,
			JobTemplate: batch_v1beta1.JobTemplateSpec{
				Spec: batch_v1.JobSpec{
					Template: corev1.PodTemplateSpec{
						Spec: podSpec,
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

func (r *ReconcileQliksense) updateOpsRunnerCronJob(cronJob *batch_v1beta1.CronJob, reqLogger logr.Logger, m *qlikv1.Qliksense) error {
	if err := r.updateJobPodSpec(&cronJob.Spec.JobTemplate.Spec.Template.Spec, reqLogger, m); err != nil {
		return err
	}
	updateJobMetadata(&cronJob.ObjectMeta, m)
	cronJob.Spec.Schedule = m.Spec.OpsRunner.Schedule
	if err := controllerutil.SetControllerReference(m, cronJob, r.scheme); err != nil {
		reqLogger.Error(err, "Error setting controller reference for cronJob")
		return err
	}
	return nil
}

func (r *ReconcileQliksense) getOpsRunnerJob(reqLogger logr.Logger, m *qlikv1.Qliksense) (*batch_v1.Job, error) {
	podSpec := corev1.PodSpec{}
	if err := r.updateJobPodSpec(&podSpec, reqLogger, m); err != nil {
		return nil, err
	}

	objectMeta := metav1.ObjectMeta{}
	updateJobMetadata(&objectMeta, m)

	job := &batch_v1.Job{
		ObjectMeta: objectMeta,
		Spec: batch_v1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: podSpec,
			},
		},
	}

	if err := controllerutil.SetControllerReference(m, job, r.scheme); err != nil {
		reqLogger.Error(err, "Error setting controller reference for job")
		return nil, err
	}
	return job, nil
}

func (r *ReconcileQliksense) updateOpsRunnerJob(job *batch_v1.Job, reqLogger logr.Logger, m *qlikv1.Qliksense) error {
	if err := r.updateJobPodSpec(&job.Spec.Template.Spec, reqLogger, m); err != nil {
		return err
	}
	updateJobMetadata(&job.ObjectMeta, m)
	if err := controllerutil.SetControllerReference(m, job, r.scheme); err != nil {
		reqLogger.Error(err, "Error setting controller reference for job")
		return err
	}
	return nil
}

func updateJobMetadata(objectMeta *metav1.ObjectMeta, m *qlikv1.Qliksense) {
	objectMeta.Name = fmt.Sprintf("%v%v", m.Name, opsRunnerJobNameSuffix)
	objectMeta.Namespace = m.Namespace
	if objectMeta.Labels == nil {
		objectMeta.Labels = make(map[string]string)
	}
	objectMeta.Labels["release"] = m.Name
}

func (r *ReconcileQliksense) updateJobPodSpec(podSpec *corev1.PodSpec, reqLogger logr.Logger, m *qlikv1.Qliksense) error {
	containerImagePullPolicy := os.Getenv("DEBUG_OPS_RUNNER_CONTAINER_IMAGE_PULL_POLICY")
	if containerImagePullPolicy == "" {
		containerImagePullPolicy = string(corev1.PullAlways)
	}
	podSpecRestartPolicy := os.Getenv("DEBUG_OPS_RUNNER_POD_SPEC_RESTART_POLICY")
	if podSpecRestartPolicy == "" {
		podSpecRestartPolicy = string(corev1.RestartPolicyOnFailure)
	}
	if len(podSpec.Containers) == 0 {
		podSpec.Containers = append(podSpec.Containers, corev1.Container{})
	}
	podSpec.Containers[0].Image = m.Spec.OpsRunner.Image
	podSpec.Containers[0].ImagePullPolicy = corev1.PullPolicy(containerImagePullPolicy)
	podSpec.Containers[0].Name = fmt.Sprintf("%v%v", m.Name, opsRunnerJobNameSuffix)

	if envVars, err := getEnvVars(podSpec, reqLogger, m); err != nil {
		return err
	} else {
		podSpec.Containers[0].Env = envVars
		reqLogger.Info("job's new podSpec.Containers[0].Env", "vars", podSpec.Containers[0].Env)
	}

	podSpec.RestartPolicy = corev1.RestartPolicy(podSpecRestartPolicy)
	podSpec.ServiceAccountName = "qliksense-operator"
	updateJobPodSpecForImageRegistry(m, podSpec)
	return nil
}

func crToYaml(m *qlikv1.Qliksense) ([]byte, error) {
	rawMap := map[string]interface{}{}
	if k8sSecretYamlBytes, err := yaml.Marshal(m); err != nil {
		return nil, err
	} else if err := yaml.Unmarshal(k8sSecretYamlBytes, &rawMap); err != nil {
		return nil, err
	} else {
		deleteMetadataProperties := []string{
			"creationTimestamp",
			"finalizers",
			"generation",
			"resourceVersion",
			"selfLink",
			"uid",
		}
		metadataMap := rawMap["metadata"].(map[string]interface{})
		for _, deleteMetadataProperty := range deleteMetadataProperties {
			delete(metadataMap, deleteMetadataProperty)
		}
		if _, ok := metadataMap["annotations"]; ok {
			annotationsMap := metadataMap["annotations"].(map[string]interface{})
			delete(annotationsMap, "kubectl.kubernetes.io/last-applied-configuration")
			if len(annotationsMap) == 0 {
				delete(metadataMap, "annotations")
			}
		}
		delete(rawMap, "status")
		return yaml.Marshal(rawMap)
	}
}

func getEnvVars(podSpec *corev1.PodSpec, reqLogger logr.Logger, m *qlikv1.Qliksense) ([]corev1.EnvVar, error) {
	operatorName, err := k8sutil.GetOperatorName()
	if err != nil {
		reqLogger.Error(err, "Error obtaining operator name")
		return nil, err
	}

	crYaml, err := crToYaml(m)
	if err != nil {
		reqLogger.Error(err, "Error marshalling CR to yaml")
		return nil, err
	}

	updateVarNames := []string{"YAML_CONF", "OPERATOR_SERVICE_NAME", "OPERATOR_SERVICE_PORT"}
	updateVarValues := map[string]string{
		"YAML_CONF":             string(crYaml),
		"OPERATOR_SERVICE_NAME": fmt.Sprintf("%s-kuztomize", operatorName),
		"OPERATOR_SERVICE_PORT": fmt.Sprintf("%v", kuzServicePort),
	}
	currentEnvVarNames := make(map[string]bool)
	currentEnvVars := podSpec.Containers[0].Env
	newEnvVars := make([]corev1.EnvVar, 0)

	for _, currentEnvVar := range currentEnvVars {
		currentEnvVarNames[currentEnvVar.Name] = true
		if updateVarValue, ok := updateVarValues[currentEnvVar.Name]; ok {
			currentEnvVar.Value = updateVarValue
		}
		newEnvVars = append(newEnvVars, currentEnvVar)
	}
	for _, updateVarName := range updateVarNames {
		if _, present := currentEnvVarNames[updateVarName]; !present {
			newEnvVars = append(newEnvVars, corev1.EnvVar{
				Name:  updateVarName,
				Value: updateVarValues[updateVarName],
			})
		}
	}
	return newEnvVars, nil
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
