package qust

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Shopify/ejson"
	"github.com/qlik-oss/qliksense-operator/pkg/config"
	"github.com/qlik-oss/qliksense-operator/pkg/keys"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
)

func TestSetupServiceSecretsDirectory(t *testing.T) {
	publicKey, _, err := ejson.GenerateKeypair()
	tmpManifestsRootDir, err := setupTests(t, "TestSetupServiceSecretsDirectory-", publicKey)
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
	publicKey, privateKey, err := ejson.GenerateKeypair()
	tmpManifestsRootDir, err := setupTests(t, "TestWriteServiceEpriviteKeyJsonFile-", publicKey)
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
	service.PrivateKey, service.Kid, service.JWKS, err = keys.Generate()
	assert.NoError(t, err)

	err = os.Mkdir(filepath.Join(tmpManifestsRootDir, operatorPatchBaseFolder, "secrets/foo"), 0777)
	assert.NoError(t, err)

	service.PrivateKey = strings.Replace(service.PrivateKey, "\n", "\\n", -1)
	err = writeServiceEpriviteKeyJsonFile(cr, service)
	assert.NoError(t, err)

	verifyServiceEpriviteKeyJsonFile(t, tmpManifestsRootDir, service.Name, privateKey)
}

func verifyServiceEpriviteKeyJsonFile(t *testing.T, tmpManifestsRootDir string, serviceName string, privateKey string) {
	ePrivateKeyJsonFilePath := filepath.Join(tmpManifestsRootDir, operatorPatchBaseFolder, fmt.Sprintf("secrets/%v/eprivate_key.json", serviceName))
	ePrivateKeyJsonFileBytes, err := ioutil.ReadFile(ePrivateKeyJsonFilePath)
	assert.NoError(t, err)

	var ePrivateKeyJsonMap map[string]interface{}
	err = json.Unmarshal(ePrivateKeyJsonFileBytes, &ePrivateKeyJsonMap)
	assert.NoError(t, err)

	_, privateKeyPropExists := ePrivateKeyJsonMap["private_key"]
	assert.True(t, privateKeyPropExists)
	assert.NotContains(t, ePrivateKeyJsonMap["private_key"], "BEGIN EC PRIVATE KEY")
	assert.NotContains(t, ePrivateKeyJsonMap["private_key"], "END EC PRIVATE KEY")

	_, kidPropExists := ePrivateKeyJsonMap["kid"]
	assert.True(t, kidPropExists)

	decryptedBytes, err := ejson.DecryptFile(ePrivateKeyJsonFilePath, "", privateKey)
	assert.NoError(t, err)

	var decryptedMap map[string]interface{}
	err = json.Unmarshal(decryptedBytes, &decryptedMap)
	assert.NoError(t, err)

	assert.Contains(t, decryptedMap["private_key"], "BEGIN EC PRIVATE KEY")
	assert.Contains(t, decryptedMap["private_key"], "END EC PRIVATE KEY")
}

func TestWriteServiceSelectivePatchYamlFile(t *testing.T) {
	publicKey, _, err := ejson.GenerateKeypair()
	tmpManifestsRootDir, err := setupTests(t, "TestWriteServiceSelectivePatchYamlFile-", publicKey)
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

	err = os.Mkdir(filepath.Join(tmpManifestsRootDir, operatorPatchBaseFolder, "secrets/foo"), 0777)
	assert.NoError(t, err)

	err = writeServiceSelectivePatchYamlFile(cr, service)
	assert.NoError(t, err)

	verifyServiceSelectivePatchYamlFile(t, tmpManifestsRootDir, service.Name)
}

func verifyServiceSelectivePatchYamlFile(t *testing.T, tmpManifestsRootDir string, serviceName string) {
	selectivePatchYamlFilePath := filepath.Join(tmpManifestsRootDir, operatorPatchBaseFolder, fmt.Sprintf("secrets/%v/selectivepatch.yaml", serviceName))
	selectivePatchYamlFileBytes, err := ioutil.ReadFile(selectivePatchYamlFilePath)
	assert.NoError(t, err)

	var selectivePatchYamlMap map[string]interface{}
	err = yaml.Unmarshal(selectivePatchYamlFileBytes, &selectivePatchYamlMap)
	assert.NoError(t, err)
	assert.Equal(t, fmt.Sprintf("%v-component-secrets-operator", serviceName), selectivePatchYamlMap["metadata"].(map[interface{}]interface{})["name"])

	var patchYamlMap map[string]interface{}
	err = yaml.Unmarshal([]byte(selectivePatchYamlMap["patches"].([]interface{})[0].(map[interface{}]interface{})["patch"].(string)), &patchYamlMap)
	assert.NoError(t, err)
	assert.Equal(t, fmt.Sprintf("%v-secrets", serviceName), patchYamlMap["metadata"].(map[interface{}]interface{})["name"])
}

func TestGenerateKeys(t *testing.T) {
	testCases := []struct {
		name                     string
		secretsKustomizationYaml string
		configsKustomizationYaml string
	}{
		{
			name: "noPreviousPatches",
			secretsKustomizationYaml: `
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
- qliksense.yaml
transformers:
- ../../transformers
`,
			configsKustomizationYaml: `
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
`,
		},
		{
			name: "withPreviousPatches",
			secretsKustomizationYaml: `
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
- qliksense.yaml
- foo
- bar
transformers:
- ../../transformers
`,
			configsKustomizationYaml: `
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
- keys
`,
		},
		{
			name: "withSomePreviousPatches",
			secretsKustomizationYaml: `
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
- qliksense.yaml
#- foo
- bar
transformers:
- ../../transformers
`,
			configsKustomizationYaml: `
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
- abra-cadabra
`,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			publicKey, privateKey, err := ejson.GenerateKeypair()
			tmpManifestsRootDir, err := setupTests(t, fmt.Sprintf("TestGenerateKeys-%v-", testCase.name), publicKey)
			if err == nil {
				defer func() {
					err := os.RemoveAll(tmpManifestsRootDir)
					if err != nil {
						fmt.Printf("Error deleting test tmp directory: %v\n", err)
					}
				}()
			}
			assert.NoError(t, err)

			secretsKustomizationYamlPath := filepath.Join(tmpManifestsRootDir, operatorPatchBaseFolder, "secrets/kustomization.yaml")
			err = ioutil.WriteFile(secretsKustomizationYamlPath, []byte(testCase.secretsKustomizationYaml), 0777)
			assert.NoError(t, err)

			configsKustomizationYamlPath := filepath.Join(tmpManifestsRootDir, operatorPatchBaseFolder, "configs/kustomization.yaml")
			err = ioutil.WriteFile(configsKustomizationYamlPath, []byte(testCase.configsKustomizationYaml), 0777)
			assert.NoError(t, err)

			cr := &config.CRConfig{
				ManifestsRoot: tmpManifestsRootDir,
			}

			err = GenerateKeys(cr)
			assert.NoError(t, err)

			verifyServicePatches(t, tmpManifestsRootDir, privateKey)
			verifySecretsAndConfigsKustomizationFiles(t, secretsKustomizationYamlPath, configsKustomizationYamlPath)
		})
	}
}

func verifySecretsAndConfigsKustomizationFiles(t *testing.T, secretsKustomizationYamlPath string, configsKustomizationYamlPath string) {
	secretsKustomizationYamlFinalBytes, err := ioutil.ReadFile(secretsKustomizationYamlPath)
	assert.NoError(t, err)

	var secretsKustomizationYamlMap map[string]interface{}
	err = yaml.Unmarshal(secretsKustomizationYamlFinalBytes, &secretsKustomizationYamlMap)
	assert.NoError(t, err)

	assert.Contains(t, secretsKustomizationYamlMap["resources"].([]interface{}), "foo")
	assert.Contains(t, secretsKustomizationYamlMap["resources"].([]interface{}), "bar")

	configsKustomizationYamlFinalBytes, err := ioutil.ReadFile(configsKustomizationYamlPath)
	assert.NoError(t, err)

	var configsKustomizationYamlMap map[string]interface{}
	err = yaml.Unmarshal(configsKustomizationYamlFinalBytes, &configsKustomizationYamlMap)
	assert.NoError(t, err)

	assert.Contains(t, configsKustomizationYamlMap["resources"].([]interface{}), "keys")
}

func verifyServicePatches(t *testing.T, tmpManifestsRootDir string, privateKey string) {
	verifyServiceEpriviteKeyJsonFile(t, tmpManifestsRootDir, "foo", privateKey)
	verifyServiceEpriviteKeyJsonFile(t, tmpManifestsRootDir, "bar", privateKey)

	verifyServiceSelectivePatchYamlFile(t, tmpManifestsRootDir, "foo")
	verifyServiceSelectivePatchYamlFile(t, tmpManifestsRootDir, "bar")

	keysConfigsKustomizationFilePath := filepath.Join(tmpManifestsRootDir, operatorPatchBaseFolder, "configs/keys/kustomization.yaml")
	keysConfigsKustomizationFileBytes, err := ioutil.ReadFile(keysConfigsKustomizationFilePath)
	assert.NoError(t, err)
	assert.Equal(t, keysConfigs_kustomizationYaml, string(keysConfigsKustomizationFileBytes))

	keysConfigsGomplateFilePath := filepath.Join(tmpManifestsRootDir, operatorPatchBaseFolder, "configs/keys/gomplate.yaml")
	keysConfigsGomplateFileBytes, err := ioutil.ReadFile(keysConfigsGomplateFilePath)
	assert.NoError(t, err)
	assert.Equal(t, keysConfigs_gomplateYaml, string(keysConfigsGomplateFileBytes))

	keysConfigsSelectivePatchYamlFilePath := filepath.Join(tmpManifestsRootDir, operatorPatchBaseFolder, "configs/keys/selectivepatch.yaml")
	keysConfigsSelectivePatchYamlFileBytes, err := ioutil.ReadFile(keysConfigsSelectivePatchYamlFilePath)
	assert.NoError(t, err)

	var selectivePatchYamlMap map[string]interface{}
	err = yaml.Unmarshal(keysConfigsSelectivePatchYamlFileBytes, &selectivePatchYamlMap)
	assert.NoError(t, err)

	var patchYamlMap map[string]interface{}
	err = yaml.Unmarshal([]byte(selectivePatchYamlMap["patches"].([]interface{})[0].(map[interface{}]interface{})["patch"].(string)), &patchYamlMap)
	assert.NoError(t, err)
	assert.Equal(t, `(( (ds "data").foo ))`, patchYamlMap["data"].(map[interface{}]interface{})["qlik.api.internal-foo"])
	assert.Equal(t, `(( (ds "data").bar ))`, patchYamlMap["data"].(map[interface{}]interface{})["qlik.api.internal-bar"])

	eJwksJsonFilePath := filepath.Join(tmpManifestsRootDir, operatorPatchBaseFolder, "configs/keys/ejwks.json")
	eJwksJsonFileBytes, err := ioutil.ReadFile(eJwksJsonFilePath)
	assert.NoError(t, err)

	var eJwksJsonMap map[string]interface{}
	err = json.Unmarshal(eJwksJsonFileBytes, &eJwksJsonMap)
	assert.NoError(t, err)
	_, fooPropExists := eJwksJsonMap["foo"]
	assert.True(t, fooPropExists)
	assert.NotContains(t, eJwksJsonMap["foo"], "BEGIN PUBLIC KEY")
	assert.NotContains(t, eJwksJsonMap["foo"], "END PUBLIC KEY")
	_, barPropExists := eJwksJsonMap["bar"]
	assert.True(t, barPropExists)
	assert.NotContains(t, eJwksJsonMap["bar"], "BEGIN PUBLIC KEY")
	assert.NotContains(t, eJwksJsonMap["bar"], "END PUBLIC KEY")

	decryptedBytes, err := ejson.DecryptFile(eJwksJsonFilePath, "", privateKey)
	assert.NoError(t, err)
	var decryptedMap map[string]interface{}
	err = json.Unmarshal(decryptedBytes, &decryptedMap)
	assert.NoError(t, err)
	assert.Contains(t, decryptedMap["foo"], "BEGIN PUBLIC KEY")
	assert.Contains(t, decryptedMap["foo"], "END PUBLIC KEY")
	assert.Contains(t, decryptedMap["bar"], "BEGIN PUBLIC KEY")
	assert.Contains(t, decryptedMap["bar"], "END PUBLIC KEY")
}

const servicesYaml = `
- name: foo
- name: bar
`

const serviceSecrets_ePrivateKeyJsonTmpl = `{
  "_public_key": "%v",
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
  "_public_key": "%v"
  {{range .}},"{{.Name}}": "{{.JWKS}}"
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

const keysConfigs_selectivepatchYamlTmpl = `
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
`

func setupTests(t *testing.T, tmpDirPrefix string, publicKey string) (tmpDir string, err error) {
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
	if err := ioutil.WriteFile(filepath.Join(tmpDir, operatorPatchBaseFolder, "templates/keys/keys_configs/ejwks.json.tmpl"), []byte(fmt.Sprintf(keysConfigs_ejwksJsonTmpl, publicKey)), 0777); err != nil {
		return "", err
	}
	if err := ioutil.WriteFile(filepath.Join(tmpDir, operatorPatchBaseFolder, "templates/keys/keys_configs/gomplate.yaml"), []byte(keysConfigs_gomplateYaml), 0777); err != nil {
		return "", err
	}
	if err := ioutil.WriteFile(filepath.Join(tmpDir, operatorPatchBaseFolder, "templates/keys/keys_configs/kustomization.yaml"), []byte(keysConfigs_kustomizationYaml), 0777); err != nil {
		return "", err
	}
	if err := ioutil.WriteFile(filepath.Join(tmpDir, operatorPatchBaseFolder, "templates/keys/keys_configs/selectivepatch.yaml.tmpl"), []byte(keysConfigs_selectivepatchYamlTmpl), 0777); err != nil {
		return "", err
	}

	if err := os.MkdirAll(filepath.Join(tmpDir, operatorPatchBaseFolder, "templates/keys/service_secrets"), 0777); err != nil {
		return "", err
	}
	if err := ioutil.WriteFile(filepath.Join(tmpDir, operatorPatchBaseFolder, "templates/keys/service_secrets/eprivate_key.json.tmpl"), []byte(fmt.Sprintf(serviceSecrets_ePrivateKeyJsonTmpl, publicKey)), 0777); err != nil {
		return "", err
	}
	if err := ioutil.WriteFile(filepath.Join(tmpDir, operatorPatchBaseFolder, "templates/keys/service_secrets/kustomization.yaml"), []byte(serviceSecrets_kustomizationYaml), 0777); err != nil {
		return "", err
	}
	if err := ioutil.WriteFile(filepath.Join(tmpDir, operatorPatchBaseFolder, "templates/keys/service_secrets/selectivepatch.yaml.tmpl"), []byte(serviceSecrets_selectivepatchYamlTmpl), 0777); err != nil {
		return "", err
	}

	if err := ioutil.WriteFile(filepath.Join(tmpDir, operatorPatchBaseFolder, "templates/keys/services.yaml"), []byte(servicesYaml), 0777); err != nil {
		return "", err
	}

	return tmpDir, err
}
