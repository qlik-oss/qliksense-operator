package qust

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/qlik-oss/qliksense-operator/pkg/keys"

	"github.com/qlik-oss/qliksense-operator/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestSetupServiceSecretsDirectory(t *testing.T) {
	tmpManifestsRootDir, err := setupTmpManifestsRootDir(t, "TestSetupServiceSecretsDirectory-")
	if err == nil {
		defer func() {
			err := os.RemoveAll(tmpManifestsRootDir)
			if err != nil {
				fmt.Printf("Error deleting test tmp directory: %v\n", err)
			}
		}()
	}
	assert.NoError(t, err)

	cr := &config.CRConfig{
		ManifestsRoot: tmpManifestsRootDir,
	}

	service := &serviceT{
		Name: "foo",
	}

	err = setupServiceSecretsDirectory(cr, service)
	assert.NoError(t, err)

	kustomizationFilePath := filepath.Join(tmpManifestsRootDir, operatorPatchBaseFolder, "secrets/foo/kustomization.yaml")
	assert.FileExists(t, kustomizationFilePath)

	kustomizationFileBytes, err := ioutil.ReadFile(kustomizationFilePath)
	assert.NoError(t, err)

	assert.Equal(t, serviceSecrets_kustomizationYaml, string(kustomizationFileBytes))
}

func TestWriteServiceEpriviteKeyJsonFile(t *testing.T) {
	tmpManifestsRootDir, err := setupTmpManifestsRootDir(t, "TestWriteServiceEpriviteKeyJsonFile-")
	if err == nil {
		defer func(){
			err := os.RemoveAll(tmpManifestsRootDir)
			if err != nil {
				fmt.Printf("Error deleting test tmp directory: %v\n", err)
			}
		}()
	}
	assert.NoError(t, err)

	cr := &config.CRConfig{
		ManifestsRoot: tmpManifestsRootDir,
	}

	service := &serviceT{
		Name: "foo",
	}
	service.PrivateKey, service.Kid, service.JWKS, err = keys.Generate()
	assert.NoError(t, err)

	err = os.Mkdir(filepath.Join(tmpManifestsRootDir, operatorPatchBaseFolder, "secrets/foo"), 0777)
	assert.NoError(t, err)

	privateKeyUnescaped := service.PrivateKey
	service.PrivateKey = strings.Replace(service.PrivateKey, "\n", "\\n", -1)
	err = writeServiceEpriviteKeyJsonFile(cr, service)
	assert.NoError(t, err)

	ePrivateKeyJsonFilePath := filepath.Join(tmpManifestsRootDir, operatorPatchBaseFolder, "secrets/foo/eprivate_key.json")
	assert.FileExists(t, ePrivateKeyJsonFilePath)

	ePrivateKeyJsonFileBytes, err := ioutil.ReadFile(ePrivateKeyJsonFilePath)
	assert.NoError(t, err)

	var ePrivateKeyJsonMap map[string]interface{}
	err = json.Unmarshal(ePrivateKeyJsonFileBytes, &ePrivateKeyJsonMap)
	assert.NoError(t, err)

	assert.Equal(t, privateKeyUnescaped, ePrivateKeyJsonMap["private_key"])
	assert.Equal(t, service.Kid, ePrivateKeyJsonMap["kid"])
}

const servicesYaml = `services:
- name: foo
- name: bar
`

const serviceSecrets_ePrivateKeyJsonTmpl = `{
  "_public_key": "04eb6c4c0b7ffec7a02a9c36919a36f94f45b90d0850ce79f559bf09a6782278",
  "private_key": "{{.PrivateKey}}",
  "kid": "{{.Kid}}"
}
`
const serviceSecrets_kustomizationYaml = `
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - selectivepatch.yaml
transformers:
  - ../../../manifests/base/transformers/gomplate
`

const serviceSecrets_selectivepatchYamlTmpl = `
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
`

const keysConfigs_ejwksJsonTmpl = `{
  "_public_key": "04eb6c4c0b7ffec7a02a9c36919a36f94f45b90d0850ce79f559bf09a6782278"
  {{range .services}}
  ,"{{.Name}}": "{{.JWKS}}"
  {{end}}
}
`

const keysConfigs_gomplateYaml = `
apiVersion: qlik.com/v1
kind: Gomplate
metadata:
  name: gomplate
  labels:
    key: gomplate
dataSource:
  ejson:
    filePath: ejwks.json
`

const keysConfigs_kustomizationYaml = `
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - selectivepatch.yaml
transformers:
  - gomplate.yaml
`

const keysConfigs_selectivepatchYaml = `
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
        {{range .services}}
        qlik.api.internal-{{.Name}}: (( (ds "data").{{.Name}} ))
        {{end}}
`

func setupTmpManifestsRootDir(t *testing.T, prefix string) (string, error) {
	tmpDir, err := ioutil.TempDir("", prefix)
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
	if err := ioutil.WriteFile(filepath.Join(tmpDir, operatorPatchBaseFolder, "templates/keys/keys_configs/ejwks.json.tmpl"), []byte(keysConfigs_ejwksJsonTmpl), 0777); err != nil {
		return "", err
	}
	if err := ioutil.WriteFile(filepath.Join(tmpDir, operatorPatchBaseFolder, "templates/keys/keys_configs/gomplate.yaml"), []byte(keysConfigs_gomplateYaml), 0777); err != nil {
		return "", err
	}
	if err := ioutil.WriteFile(filepath.Join(tmpDir, operatorPatchBaseFolder, "templates/keys/keys_configs/selectivepatch.yaml"), []byte(keysConfigs_selectivepatchYaml), 0777); err != nil {
		return "", err
	}

	if err := os.MkdirAll(filepath.Join(tmpDir, operatorPatchBaseFolder, "templates/keys/service_secrets"), 0777); err != nil {
		return "", err
	}
	if err := ioutil.WriteFile(filepath.Join(tmpDir, operatorPatchBaseFolder, "templates/keys/service_secrets/eprivate_key.json.tmpl"), []byte(serviceSecrets_ePrivateKeyJsonTmpl), 0777); err != nil {
		return "", err
	}
	if err := ioutil.WriteFile(filepath.Join(tmpDir, operatorPatchBaseFolder, "templates/keys/service_secrets/kustomization.yaml"), []byte(serviceSecrets_kustomizationYaml), 0777); err != nil {
		return "", err
	}
	if err := ioutil.WriteFile(filepath.Join(tmpDir, operatorPatchBaseFolder, "templates/keys/service_secrets/selectivepatch.yaml.tmpl"), []byte(serviceSecrets_selectivepatchYamlTmpl), 0777); err != nil {
		return "", err
	}

	if err := ioutil.WriteFile(filepath.Join(tmpDir, operatorPatchBaseFolder, "templates/services.yaml"), []byte(servicesYaml), 0777); err != nil {
		return "", err
	}

	return tmpDir, err
}
