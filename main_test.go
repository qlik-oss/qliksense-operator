package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/Shopify/ejson"
	"github.com/qlik-oss/qliksense-operator/pkg/config"
	"github.com/qlik-oss/qliksense-operator/pkg/qust"
	"github.com/stretchr/testify/assert"
)

type mainTestCaseT struct {
	name               string
	ejsonKeydirEnv     string
	defaultEjsonKeydir string
	verify             func(t *testing.T, tmpManifestsRootDir string)
}

func Test_generateKeys(t *testing.T) {
	testCases := []mainTestCaseT{
		func() mainTestCaseT {
			ejsonKeydirEnv, err := ioutil.TempDir("", "")
			if err != nil {
				panic(err)
			}

			defaultEjsonKeydir, err := ioutil.TempDir("", "")
			if err != nil {
				panic(err)
			}

			return mainTestCaseT{
				name:               "EJSON_KEYDIR-env-var-set",
				ejsonKeydirEnv:     ejsonKeydirEnv,
				defaultEjsonKeydir: defaultEjsonKeydir,
				verify: func(t *testing.T, tmpManifestsRootDir string) {
					pubKey, privKey, err := getEjsonKeyPair(ejsonKeydirEnv)
					assert.NoError(t, err)

					verifyEncryption(t, tmpManifestsRootDir, pubKey, privKey)
				},
			}
		}(),
		func() mainTestCaseT {
			defaultEjsonKeydir, err := ioutil.TempDir("", "")
			if err != nil {
				panic(err)
			}

			return mainTestCaseT{
				name:               "EJSON_KEYDIR-env-var-NOT-set",
				ejsonKeydirEnv:     "",
				defaultEjsonKeydir: defaultEjsonKeydir,
				verify: func(t *testing.T, tmpManifestsRootDir string) {
					pubKey, privKey, err := getEjsonKeyPair(defaultEjsonKeydir)
					assert.NoError(t, err)

					verifyEncryption(t, tmpManifestsRootDir, pubKey, privKey)
				},
			}
		}(),
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			tmpManifestsRootDir, err := qust.SetupTests(fmt.Sprintf("Test_generateKeys-%v-", testCase.name))
			if err == nil {
				defer func() {
					err := os.RemoveAll(tmpManifestsRootDir)
					if err != nil {
						fmt.Printf("Error deleting test tmp directory: %v\n", err)
					}
				}()
			}
			assert.NoError(t, err)

			secretsKustomizationYamlPath := filepath.Join(tmpManifestsRootDir, ".operator/secrets/kustomization.yaml")
			err = ioutil.WriteFile(secretsKustomizationYamlPath, []byte(`
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
- qliksense.yaml
transformers:
- ../../transformers
`), 0777)
			assert.NoError(t, err)

			configsKustomizationYamlPath := filepath.Join(tmpManifestsRootDir, ".operator/configs/kustomization.yaml")
			err = ioutil.WriteFile(configsKustomizationYamlPath, []byte(`
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
`), 0777)
			assert.NoError(t, err)

			os.Setenv("EJSON_KEYDIR", testCase.ejsonKeydirEnv)
			generateKeys(&config.CRConfig{ManifestsRoot: tmpManifestsRootDir}, testCase.defaultEjsonKeydir)
			os.Unsetenv("EJSON_KEYDIR")

			assert.NoError(t, err)
			testCase.verify(t, tmpManifestsRootDir)
		})
	}
}

func getEjsonKeyPair(dir string) (pubKey, privKey string, err error) {
	list, err := ioutil.ReadDir(dir)
	if err != nil {
		return "", "", err
	} else if len(list) != 1 {
		return "", "", errors.New("test ejson keydir should have only 1 file")
	}

	pubKey = list[0].Name()
	privKeyBytes, err := ioutil.ReadFile(path.Join(dir, pubKey))
	if err != nil {
		return "", "", err
	}
	privKey = string(privKeyBytes)
	return pubKey, privKey, err
}

func verifyEncryption(t *testing.T, tmpManifestsRootDir string, pubKey string, privKey string) {
	eJwksJsonFilePath := filepath.Join(tmpManifestsRootDir, ".operator/configs/keys/ejwks.json")

	eJwksJsonFileBytes, err := ioutil.ReadFile(eJwksJsonFilePath)
	assert.NoError(t, err)

	var eJwksJsonMap map[string]interface{}
	err = json.Unmarshal(eJwksJsonFileBytes, &eJwksJsonMap)
	assert.NoError(t, err)
	assert.Equal(t, pubKey, eJwksJsonMap["_public_key"])

	decryptedBytes, err := ejson.DecryptFile(eJwksJsonFilePath, "", privKey)
	assert.NoError(t, err)

	var decryptedMap map[string]interface{}
	err = json.Unmarshal(decryptedBytes, &decryptedMap)
	assert.NoError(t, err)
	assert.Equal(t, pubKey, eJwksJsonMap["_public_key"])
}
