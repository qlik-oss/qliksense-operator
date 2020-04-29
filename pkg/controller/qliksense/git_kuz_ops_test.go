package qliksense

import (
	"testing"

	kapis_config "github.com/qlik-oss/k-apis/pkg/config"
	"gopkg.in/yaml.v2"
)

func Test_If_condiitions(t *testing.T) {
	spec := &kapis_config.CRSpec{}
	crspec := `
git:
  repository: "https://github.com/my-org/qliksense-k8s"
  accessToken: "blalalalal"
opsRunner:
  enabled: "yes" 
  image: pre-packaged-container:v1.21.23
  crPvc: enabled
`

	if err := yaml.Unmarshal([]byte(crspec), spec); err != nil {
		t.Error(err)
	}
	if IsOnlyRunner(spec) {
		t.Log("must not only runner")
		t.Fail()
	}
	if IsOnlyGit(spec) {
		t.Log("must not only git")
		t.Fail()
	}
	if !IsGitRunner(spec) {
		t.Log("must be git runner")
		t.Fail()
	}

	crspec = `
git:
  repository: "https://github.com/my-org/qliksense-k8s"
  accessToken: "blalalalal"
opsRunner:
  enabled: "yes" 
  image: pre-packaged-container:v1.21.23
  schedule: "* * * 5/10 *"
  watchBranch: master
  crPvc: enabled
`
	spec = &kapis_config.CRSpec{}
	if err := yaml.Unmarshal([]byte(crspec), spec); err != nil {
		t.Error(err)
	}

	if !IsGitOpsEnabled(spec) {
		t.Log("must be gitops enabled")
		t.Fail()
	}
	if IsOnlyRunner(spec) {
		t.Log("must not only runner")
		t.Fail()
	}
	if IsOnlyGit(spec) {
		t.Log("must not only git")
		t.Fail()
	}
	if IsGitRunner(spec) {
		t.Log("must not be git runner")
		t.Fail()
	}

}
