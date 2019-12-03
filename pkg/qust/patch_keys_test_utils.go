package qust

import (
	"io/ioutil"
	"os"
	"path/filepath"
)

var TestConfigHarness = struct {
	ServicesYaml   string
	ServiceSecrets struct {
		EPrivateKeyJsonTmpl    string
		KustomizationYaml      string
		SelectivepatchYamlTmpl string
	}
	KeysConfigs struct {
		EJwksJsonTmpl          string
		GomplateYaml           string
		KustomizationYaml      string
		SelectivepatchYamlTmpl string
	}
}{
	ServicesYaml: `
- name: foo
- name: bar
`,
	ServiceSecrets: struct {
		EPrivateKeyJsonTmpl    string
		KustomizationYaml      string
		SelectivepatchYamlTmpl string
	}{
		EPrivateKeyJsonTmpl: `{
  "_public_key": "{{.EjsonPublicKey}}",
  "private_key": "{{.PrivateKey}}",
  "kid": "{{.Kid}}"
}
`,
		KustomizationYaml: `
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - selectivepatch.yaml
transformers:
  - ../../../manifests/base/transformers/gomplate
`,
		SelectivepatchYamlTmpl: `
apiVersion: qlik.com/v1
kind: SelectivePatch
metadata:
  name: {{.Name}}-component-secrets-operator
enabled: true
patches:
- target:
    kind: SuperSecret
  patch: |-
    apiVersion: qlik.com/v1
    kind: SuperSecret
    metadata:
      name: {{.Name}}-secrets
    stringData:
      tokenAuthPrivateKeyId: (( (ds "data").kid ))
      tokenAuthPrivateKey: |
        ((- "\n"))(( (ds "data").private_key | indent 8 ))
`,
	},
	KeysConfigs: struct {
		EJwksJsonTmpl          string
		GomplateYaml           string
		KustomizationYaml      string
		SelectivepatchYamlTmpl string
	}{
		EJwksJsonTmpl: `{
  "_public_key": "{{.EjsonPublicKey}}"
  {{range .Services}},"{{.Name}}": "{{.JWKS}}"
  {{end}}
}
`,
		GomplateYaml: `
apiVersion: qlik.com/v1
kind: Gomplate
metadata:
  name: gomplate
  labels:
    key: gomplate
dataSource:
  ejson:
    filePath: ejwks.json
`,
		KustomizationYaml: `
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - selectivepatch.yaml
transformers:
  - gomplate.yaml
`,
		SelectivepatchYamlTmpl: `
apiVersion: qlik.com/v1
kind: SelectivePatch
metadata:
  name: keys-operator-configs
enabled: true
patches:
  - target:
      labelSelector: app=keys
      kind: SuperConfigMap
    patch: |-
      apiVersion: qlik.com/v1
      kind: SuperConfigMap
      metadata:
        name: keys-configs
      data:
        {{range .}}qlik.api.internal-{{.Name}}: (( (ds "data").{{.Name}} ))
        {{end}}
`,
	},
}

func SetupTests(tmpDirPrefix string) (tmpDir string, err error) {
	tmpDir, err = ioutil.TempDir("", tmpDirPrefix)
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(filepath.Join(tmpDir, operatorPatchBaseFolder, "configs"), 0777); err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Join(tmpDir, operatorPatchBaseFolder, "secrets"), 0777); err != nil {
		return "", err
	}

	if err := os.MkdirAll(filepath.Join(tmpDir, operatorPatchBaseFolder, "templates/keys/keys_configs"), 0777); err != nil {
		return "", err
	}
	if err := ioutil.WriteFile(filepath.Join(tmpDir, operatorPatchBaseFolder, "templates/keys/keys_configs/ejwks.json.tmpl"), []byte(TestConfigHarness.KeysConfigs.EJwksJsonTmpl), 0777); err != nil {
		return "", err
	}
	if err := ioutil.WriteFile(filepath.Join(tmpDir, operatorPatchBaseFolder, "templates/keys/keys_configs/gomplate.yaml"), []byte(TestConfigHarness.KeysConfigs.GomplateYaml), 0777); err != nil {
		return "", err
	}
	if err := ioutil.WriteFile(filepath.Join(tmpDir, operatorPatchBaseFolder, "templates/keys/keys_configs/kustomization.yaml"), []byte(TestConfigHarness.KeysConfigs.KustomizationYaml), 0777); err != nil {
		return "", err
	}
	if err := ioutil.WriteFile(filepath.Join(tmpDir, operatorPatchBaseFolder, "templates/keys/keys_configs/selectivepatch.yaml.tmpl"), []byte(TestConfigHarness.KeysConfigs.SelectivepatchYamlTmpl), 0777); err != nil {
		return "", err
	}

	if err := os.MkdirAll(filepath.Join(tmpDir, operatorPatchBaseFolder, "templates/keys/service_secrets"), 0777); err != nil {
		return "", err
	}
	if err := ioutil.WriteFile(filepath.Join(tmpDir, operatorPatchBaseFolder, "templates/keys/service_secrets/eprivate_key.json.tmpl"), []byte(TestConfigHarness.ServiceSecrets.EPrivateKeyJsonTmpl), 0777); err != nil {
		return "", err
	}
	if err := ioutil.WriteFile(filepath.Join(tmpDir, operatorPatchBaseFolder, "templates/keys/service_secrets/kustomization.yaml"), []byte(TestConfigHarness.ServiceSecrets.KustomizationYaml), 0777); err != nil {
		return "", err
	}
	if err := ioutil.WriteFile(filepath.Join(tmpDir, operatorPatchBaseFolder, "templates/keys/service_secrets/selectivepatch.yaml.tmpl"), []byte(TestConfigHarness.ServiceSecrets.SelectivepatchYamlTmpl), 0777); err != nil {
		return "", err
	}

	if err := ioutil.WriteFile(filepath.Join(tmpDir, operatorPatchBaseFolder, "templates/keys/services.yaml"), []byte(TestConfigHarness.ServicesYaml), 0777); err != nil {
		return "", err
	}

	return tmpDir, err
}
