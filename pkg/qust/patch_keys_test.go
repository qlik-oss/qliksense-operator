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
	tmpManifestsRootDir, err := SetupTests("TestSetupServiceSecretsDirectory-")
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

	assert.Equal(t, TestConfigHarness.ServiceSecrets.KustomizationYaml, string(kustomizationFileBytes))
}

func TestWriteServiceEpriviteKeyJsonFile(t *testing.T) {
	publicKey, privateKey, err := ejson.GenerateKeypair()
	tmpManifestsRootDir, err := SetupTests("TestWriteServiceEpriviteKeyJsonFile-")
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
	err = writeServiceEpriviteKeyJsonFile(cr, service, publicKey)
	assert.NoError(t, err)

	verifyServiceEpriviteKeyJsonFile(t, tmpManifestsRootDir, service.Name, publicKey, privateKey)
}

func verifyServiceEpriviteKeyJsonFile(t *testing.T, tmpManifestsRootDir, serviceName, ejsonPublicKey, ejsonPrivateKey string) {
	ePrivateKeyJsonFilePath := filepath.Join(tmpManifestsRootDir, operatorPatchBaseFolder, fmt.Sprintf("secrets/%v/eprivate_key.json", serviceName))
	ePrivateKeyJsonFileBytes, err := ioutil.ReadFile(ePrivateKeyJsonFilePath)
	assert.NoError(t, err)

	var ePrivateKeyJsonMap map[string]interface{}
	err = json.Unmarshal(ePrivateKeyJsonFileBytes, &ePrivateKeyJsonMap)
	assert.NoError(t, err)

	assert.Equal(t, ejsonPublicKey, ePrivateKeyJsonMap["_public_key"])

	_, privateKeyPropExists := ePrivateKeyJsonMap["private_key"]
	assert.True(t, privateKeyPropExists)
	assert.NotContains(t, ePrivateKeyJsonMap["private_key"], "BEGIN EC PRIVATE KEY")
	assert.NotContains(t, ePrivateKeyJsonMap["private_key"], "END EC PRIVATE KEY")

	_, kidPropExists := ePrivateKeyJsonMap["kid"]
	assert.True(t, kidPropExists)

	decryptedBytes, err := ejson.DecryptFile(ePrivateKeyJsonFilePath, "", ejsonPrivateKey)
	assert.NoError(t, err)

	var decryptedMap map[string]interface{}
	err = json.Unmarshal(decryptedBytes, &decryptedMap)
	assert.NoError(t, err)

	assert.Contains(t, decryptedMap["private_key"], "BEGIN EC PRIVATE KEY")
	assert.Contains(t, decryptedMap["private_key"], "END EC PRIVATE KEY")
}

func TestWriteServiceSelectivePatchYamlFile(t *testing.T) {
	tmpManifestsRootDir, err := SetupTests("TestWriteServiceSelectivePatchYamlFile-")
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
			ejsonPublicKey, ejsonPrivateKey, err := ejson.GenerateKeypair()
			tmpManifestsRootDir, err := SetupTests(fmt.Sprintf("TestGenerateKeys-%v-", testCase.name))
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

			err = GenerateKeys(cr, ejsonPublicKey)
			assert.NoError(t, err)

			verifyServicePatches(t, tmpManifestsRootDir, ejsonPublicKey, ejsonPrivateKey)
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

func verifyServicePatches(t *testing.T, tmpManifestsRootDir, ejsonPublicKey string, ejsonPrivateKey string) {
	verifyServiceEpriviteKeyJsonFile(t, tmpManifestsRootDir, "foo", ejsonPublicKey, ejsonPrivateKey)
	verifyServiceEpriviteKeyJsonFile(t, tmpManifestsRootDir, "bar", ejsonPublicKey, ejsonPrivateKey)

	verifyServiceSelectivePatchYamlFile(t, tmpManifestsRootDir, "foo")
	verifyServiceSelectivePatchYamlFile(t, tmpManifestsRootDir, "bar")

	keysConfigsKustomizationFilePath := filepath.Join(tmpManifestsRootDir, operatorPatchBaseFolder, "configs/keys/kustomization.yaml")
	keysConfigsKustomizationFileBytes, err := ioutil.ReadFile(keysConfigsKustomizationFilePath)
	assert.NoError(t, err)
	assert.Equal(t, TestConfigHarness.KeysConfigs.KustomizationYaml, string(keysConfigsKustomizationFileBytes))

	keysConfigsGomplateFilePath := filepath.Join(tmpManifestsRootDir, operatorPatchBaseFolder, "configs/keys/gomplate.yaml")
	keysConfigsGomplateFileBytes, err := ioutil.ReadFile(keysConfigsGomplateFilePath)
	assert.NoError(t, err)
	assert.Equal(t, TestConfigHarness.KeysConfigs.GomplateYaml, string(keysConfigsGomplateFileBytes))

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
	assert.Equal(t, ejsonPublicKey, eJwksJsonMap["_public_key"])

	_, fooPropExists := eJwksJsonMap["foo"]
	assert.True(t, fooPropExists)
	assert.NotContains(t, eJwksJsonMap["foo"], "BEGIN PUBLIC KEY")
	assert.NotContains(t, eJwksJsonMap["foo"], "END PUBLIC KEY")
	_, barPropExists := eJwksJsonMap["bar"]
	assert.True(t, barPropExists)
	assert.NotContains(t, eJwksJsonMap["bar"], "BEGIN PUBLIC KEY")
	assert.NotContains(t, eJwksJsonMap["bar"], "END PUBLIC KEY")

	decryptedBytes, err := ejson.DecryptFile(eJwksJsonFilePath, "", ejsonPrivateKey)
	assert.NoError(t, err)
	var decryptedMap map[string]interface{}
	err = json.Unmarshal(decryptedBytes, &decryptedMap)
	assert.NoError(t, err)
	assert.Equal(t, ejsonPublicKey, eJwksJsonMap["_public_key"])

	assert.Contains(t, decryptedMap["foo"], "BEGIN PUBLIC KEY")
	assert.Contains(t, decryptedMap["foo"], "END PUBLIC KEY")
	assert.Contains(t, decryptedMap["bar"], "BEGIN PUBLIC KEY")
	assert.Contains(t, decryptedMap["bar"], "END PUBLIC KEY")
}
