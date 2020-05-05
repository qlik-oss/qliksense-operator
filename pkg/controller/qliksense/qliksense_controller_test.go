package qliksense

import (
	"fmt"
	"path"
	"testing"

	"k8s.io/apimachinery/pkg/runtime/schema"

	batch_v1beta1 "k8s.io/api/batch/v1beta1"

	qlikv1 "github.com/qlik-oss/qliksense-operator/pkg/apis/qlik/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/yaml"
)

type qliksenseControllerTestCase struct {
	name   string
	cr     string
	verify func(t *testing.T, cronJob *batch_v1beta1.CronJob)
}

func Test_cronJobForGitOps(t *testing.T) {
	var testCases = []qliksenseControllerTestCase{
		{
			name: "private imageRegistry NOT set in CR",
			cr: `
apiVersion: qlik.com/v1
kind: Qliksense
metadata:
  name: qlik-default
  labels:
    version: v0.0.2
spec:
  profile: docker-desktop
  git:
    repository: https://github.com/my-org/qliksense-k8s
    accessToken: balallafafafaf
  opsRunner:
    enabled: "yes"
    schedule: "*/10 * * * *"
    watchBranch: master
    image: qlik-docker-oss.bintray.io/qliksense-repo-watcher
`,
			verify: func(t *testing.T, cronJob *batch_v1beta1.CronJob) {
				cronJobImage := cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Image
				expectedImage := "qlik-docker-oss.bintray.io/qliksense-repo-watcher"
				if cronJobImage != expectedImage {
					t.Fatalf("expected cron job image to be: %v, but got: %v", expectedImage, cronJobImage)
				}

				if len(cronJob.Spec.JobTemplate.Spec.Template.Spec.ImagePullSecrets) > 0 {
					t.Fatal("expected there to be no imagePullSecrets")
				}
			},
		},
		func() qliksenseControllerTestCase {
			return qliksenseControllerTestCase{
				name: "private imageRegistry set in CR, but original image is empty",
				cr: `
apiVersion: qlik.com/v1
kind: Qliksense
metadata:
 name: qlik-default
 labels:
    version: v0.0.2
spec:
  profile: docker-desktop
  git:
    repository: https://github.com/my-org/qliksense-k8s
    accessToken: balallafafafaf
  opsRunner:
    enabled: "yes"
    schedule: "*/10 * * * *"
    watchBranch: master
  configs:
    qliksense:
    - name: imageRegistry
      value: whatever
  rotateKeys: "no"
`,
				verify: func(t *testing.T, cronJob *batch_v1beta1.CronJob) {
					cronJobImage := cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Image
					if cronJobImage != "" {
						t.Fatal("expected cron job image to be blank, but it wasn't")
					}

					if len(cronJob.Spec.JobTemplate.Spec.Template.Spec.ImagePullSecrets) > 0 {
						t.Fatal("expected there to be no imagePullSecrets")
					}
				},
			}
		}(),
		func() qliksenseControllerTestCase {
			imageRegistry := "fooRegistry"
			return qliksenseControllerTestCase{
				name: "private imageRegistry set in CR, original image had default (empty) registry",
				cr: fmt.Sprintf(`
apiVersion: qlik.com/v1
kind: Qliksense
metadata:
 name: qlik-default
 labels:
    version: v0.0.2
spec:
  profile: docker-desktop
  git:
    repository: https://github.com/my-org/qliksense-k8s
    accessToken: balallafafafaf
  opsRunner:
    enabled: "yes"
    schedule: "*/10 * * * *"
    watchBranch: master
    image: qliksense-repo-watcher
  configs:
    qliksense:
    - name: imageRegistry
      value: %v
  rotateKeys: "no"
`, imageRegistry),
				verify: func(t *testing.T, cronJob *batch_v1beta1.CronJob) {
					cronJobImage := cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Image
					expectedImage := fmt.Sprintf("%v/qliksense-repo-watcher", imageRegistry)
					if cronJobImage != expectedImage {
						t.Fatalf("expected cron job image to be: %v, but got: %v", expectedImage, cronJobImage)
					}

					if len(cronJob.Spec.JobTemplate.Spec.Template.Spec.ImagePullSecrets) != 1 {
						t.Fatal("expected there to be imagePullSecrets")
					} else if cronJob.Spec.JobTemplate.Spec.Template.Spec.ImagePullSecrets[0].Name != "artifactory-docker-secret" {
						t.Fatalf("expected to find artifactory-docker-secret imagePullSecret, but the name was: %v",
							cronJob.Spec.JobTemplate.Spec.Template.Spec.ImagePullSecrets[0].Name)
					}
				},
			}
		}(),
		func() qliksenseControllerTestCase {
			imageRegistry := "fooRegistry"
			return qliksenseControllerTestCase{
				name: "private imageRegistry set in CR, original image had specific registry",
				cr: fmt.Sprintf(`
apiVersion: qlik.com/v1
kind: Qliksense
metadata:
 name: qlik-default
 labels:
    version: v0.0.2
spec:
  profile: docker-desktop
  git:
    repository: https://github.com/my-org/qliksense-k8s
    accessToken: balallafafafaf
  opsRunner:
    enabled: "yes"
    schedule: "*/10 * * * *"
    watchBranch: master
    image: qlik-docker-oss.bintray.io/qliksense-repo-watcher
  configs:
    qliksense:
    - name: imageRegistry
      value: %v
  rotateKeys: "no"
`, imageRegistry),
				verify: func(t *testing.T, cronJob *batch_v1beta1.CronJob) {
					cronJobImage := cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Image
					expectedImage := fmt.Sprintf("%v/qliksense-repo-watcher", imageRegistry)
					if cronJobImage != expectedImage {
						t.Fatalf("expected cron job image to be: %v, but got: %v", expectedImage, cronJobImage)
					}

					if len(cronJob.Spec.JobTemplate.Spec.Template.Spec.ImagePullSecrets) != 1 {
						t.Fatal("expected there to be imagePullSecrets")
					} else if cronJob.Spec.JobTemplate.Spec.Template.Spec.ImagePullSecrets[0].Name != "artifactory-docker-secret" {
						t.Fatalf("expected to find artifactory-docker-secret imagePullSecret, but the name was: %v",
							cronJob.Spec.JobTemplate.Spec.Template.Spec.ImagePullSecrets[0].Name)
					}
				},
			}
		}(),
		func() qliksenseControllerTestCase {
			imageRegistry := "fooRegistry/some/path"
			return qliksenseControllerTestCase{
				name: "private imageRegistry with path set in CR",
				cr: fmt.Sprintf(`
apiVersion: qlik.com/v1
kind: Qliksense
metadata:
 name: qlik-default
 labels:
    version: v0.0.2
spec:
  profile: docker-desktop
  git:
    repository: https://github.com/my-org/qliksense-k8s
    accessToken: balallafafafaf
  opsRunner:
    enabled: "yes"
    schedule: "*/10 * * * *"
    watchBranch: master
    image: qlik-docker-oss.bintray.io/qliksense-repo-watcher
  configs:
    qliksense:
    - name: imageRegistry
      value: %v
  rotateKeys: "no"
`, imageRegistry),
				verify: func(t *testing.T, cronJob *batch_v1beta1.CronJob) {
					cronJobImage := cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Image
					expectedImage := path.Join(imageRegistry, "qliksense-repo-watcher")
					if cronJobImage != expectedImage {
						t.Fatalf("expected cron job image to be: %v, but got: %v", expectedImage, cronJobImage)
					}

					if len(cronJob.Spec.JobTemplate.Spec.Template.Spec.ImagePullSecrets) != 1 {
						t.Fatal("expected there to be imagePullSecrets")
					} else if cronJob.Spec.JobTemplate.Spec.Template.Spec.ImagePullSecrets[0].Name != "artifactory-docker-secret" {
						t.Fatalf("expected to find artifactory-docker-secret imagePullSecret, but the name was: %v",
							cronJob.Spec.JobTemplate.Spec.Template.Spec.ImagePullSecrets[0].Name)
					}
				},
			}
		}(),
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			m := &qlikv1.Qliksense{}

			scheme := runtime.NewScheme()
			scheme.AddKnownTypeWithName(schema.GroupVersion{Group: "qlik.com", Version: "v1"}.WithKind("Qliksense"), m)
			reconcileQliksense := &ReconcileQliksense{
				scheme: scheme,
			}
			reqLogger := log.WithName("test")

			if err := yaml.Unmarshal([]byte(testCase.cr), m); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			cronJob, err := reconcileQliksense.getOpsRunnerCronJob(reqLogger, m)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			testCase.verify(t, cronJob)
		})
	}
}
