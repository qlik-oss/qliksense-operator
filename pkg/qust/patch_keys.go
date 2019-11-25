package qust

import (
	"bytes"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/Shopify/ejson"
	"github.com/qlik-oss/qliksense-operator/pkg/config"
	"github.com/qlik-oss/qliksense-operator/pkg/keys"
	"gopkg.in/yaml.v2"
)

type serviceT struct {
	Name       string `yaml:"name,omitempty"`
	PrivateKey string
	Kid        string
	JWKS       string
}

func GenerateKeys(cr *config.CRConfig) error {
	serviceList, err := initServiceList(cr)
	if err != nil {
		return err
	}
	for _, service := range serviceList {
		if service.PrivateKey, service.Kid, service.JWKS, err = keys.Generate(); err != nil {
			return err
		}

		//escaping for valid insertion into JSON via templating:
		service.PrivateKey = strings.Replace(service.PrivateKey, "\n", `\n`, -1)
		service.JWKS = strings.Replace(service.JWKS, `"`, `\"`, -1)

		if err := writeServiceSecrets(cr, service); err != nil {
			return err
		}
	}
	if err := writeKeysConfigs(cr, serviceList); err != nil {
		return err
	}
	if err := overrideSecretsKustomizationYaml(cr, serviceList); err != nil {
		return err
	}
	if err := overrideConfigsKustomizationYaml(cr); err != nil {
		return err
	}

	return nil
}

func initServiceList(cr *config.CRConfig) ([]*serviceT, error) {
	serviceListYamlPath := filepath.Join(cr.ManifestsRoot, operatorPatchBaseFolder, "templates/keys/services.yaml")
	yamlBytes, err := ioutil.ReadFile(serviceListYamlPath)
	if err != nil {
		return nil, err
	}

	var serviceList []*serviceT
	err = yaml.Unmarshal(yamlBytes, &serviceList)
	if err != nil {
		return nil, err
	}

	return serviceList, nil
}

func writeServiceSecrets(cr *config.CRConfig, service *serviceT) error {
	if err := setupServiceSecretsDirectory(cr, service); err != nil {
		return err
	}
	if err := writeServiceEpriviteKeyJsonFile(cr, service); err != nil {
		return err
	}
	if err := writeServiceSelectivePatchYamlFile(cr, service); err != nil {
		return err
	}
	return nil
}

func writeKeysConfigs(cr *config.CRConfig, serviceList []*serviceT) error {
	if err := setupKeysConfigsDirectory(cr); err != nil {
		return err
	}
	if err := writeKeysEjwksJsonFile(cr, serviceList); err != nil {
		return err
	}
	if err := writeKeysSelectivePatchYamlFile(cr, serviceList); err != nil {
		return err
	}
	return nil
}

func overrideSecretsKustomizationYaml(cr *config.CRConfig, services []*serviceT) error {
	kustomizationYamlPath := filepath.Join(cr.ManifestsRoot, operatorPatchBaseFolder, "secrets/kustomization.yaml")
	return smallFileCopy(kustomizationYamlPath, kustomizationYamlPath, 0777, func(in []byte) ([]byte, error) {
		var yamlMap map[string]interface{}
		if err := yaml.Unmarshal(in, &yamlMap); err != nil {
			return nil, err
		}

		presentResources := make([]interface{}, 0)
		if yamlMap["resources"] != nil {
			presentResources = yamlMap["resources"].([]interface{})
		}

		presentResourcesMap := make(map[string]bool)
		for _, resource := range presentResources {
			presentResourcesMap[resource.(string)] = true
		}

		for _, service := range services {
			if _, ok := presentResourcesMap[service.Name]; !ok {
				presentResources = append(presentResources, service.Name)
			}
		}

		yamlMap["resources"] = presentResources
		out, err := yaml.Marshal(&yamlMap)
		if err != nil {
			return nil, err
		}

		return out, nil
	})
}

func overrideConfigsKustomizationYaml(cr *config.CRConfig) error {
	kustomizationYamlPath := filepath.Join(cr.ManifestsRoot, operatorPatchBaseFolder, "configs/kustomization.yaml")
	return smallFileCopy(kustomizationYamlPath, kustomizationYamlPath, 0777, func(in []byte) ([]byte, error) {
		var yamlMap map[string]interface{}
		if err := yaml.Unmarshal(in, &yamlMap); err != nil {
			return nil, err
		}

		presentResources := make([]interface{}, 0)
		if yamlMap["resources"] != nil {
			presentResources = yamlMap["resources"].([]interface{})
		}

		foundKeysResource := false
		for _, resource := range presentResources {
			if resource.(string) == "keys" {
				foundKeysResource = true
				break
			}
		}
		if !foundKeysResource {
			presentResources = append(presentResources, "keys")
		}

		yamlMap["resources"] = presentResources
		out, err := yaml.Marshal(&yamlMap)
		if err != nil {
			return nil, err
		}

		return out, nil
	})
}

func setupServiceSecretsDirectory(cr *config.CRConfig, service *serviceT) error {
	serviceSecretsDirPath := filepath.Join(cr.ManifestsRoot, operatorPatchBaseFolder, "secrets", service.Name)
	if err := os.RemoveAll(serviceSecretsDirPath); err != nil {
		return err
	}
	if err := os.Mkdir(serviceSecretsDirPath, 0777); err != nil {
		return err
	}
	if err := smallFileCopy(
		filepath.Join(cr.ManifestsRoot, operatorPatchBaseFolder, "templates/keys/service_secrets/kustomization.yaml"),
		filepath.Join(cr.ManifestsRoot, operatorPatchBaseFolder, "secrets", service.Name, "kustomization.yaml"), 0777, nil); err != nil {
		return err
	}
	return nil
}

func writeServiceEpriviteKeyJsonFile(cr *config.CRConfig, service *serviceT) error {
	templateFilePath := filepath.Join(cr.ManifestsRoot, operatorPatchBaseFolder, "templates/keys/service_secrets/eprivate_key.json.tmpl")
	destinationFilePath := filepath.Join(cr.ManifestsRoot, operatorPatchBaseFolder, "secrets", service.Name, "eprivate_key.json")
	var byteBuffer bytes.Buffer
	if template, err := template.New(path.Base(templateFilePath)).ParseFiles(templateFilePath); err != nil {
		return err
	} else if err := template.Execute(&byteBuffer, service); err != nil {
		return err
	} else if file, err := os.Create(destinationFilePath); err != nil {
		return err
	} else if _, err := ejson.Encrypt(&byteBuffer, file); err != nil {
		return err
	}
	return nil
}

func writeServiceSelectivePatchYamlFile(cr *config.CRConfig, service *serviceT) error {
	templateFilePath := filepath.Join(cr.ManifestsRoot, operatorPatchBaseFolder, "templates/keys/service_secrets/selectivepatch.yaml.tmpl")
	destinationFilePath := filepath.Join(cr.ManifestsRoot, operatorPatchBaseFolder, "secrets", service.Name, "selectivepatch.yaml")
	if template, err := template.New(path.Base(templateFilePath)).ParseFiles(templateFilePath); err != nil {
		return err
	} else if file, err := os.Create(destinationFilePath); err != nil {
		return err
	} else if err := template.Execute(file, service); err != nil {
		return err
	}
	return nil
}

func setupKeysConfigsDirectory(cr *config.CRConfig) error {
	configsDirPath := filepath.Join(cr.ManifestsRoot, operatorPatchBaseFolder, "configs/keys")
	if err := os.RemoveAll(configsDirPath); err != nil {
		return err
	}
	if err := os.Mkdir(configsDirPath, 0777); err != nil {
		return err
	}
	if err := smallFileCopy(
		filepath.Join(cr.ManifestsRoot, operatorPatchBaseFolder, "templates/keys/keys_configs/kustomization.yaml"),
		filepath.Join(cr.ManifestsRoot, operatorPatchBaseFolder, "configs/keys/kustomization.yaml"), 0777, nil); err != nil {
		return err
	}
	if err := smallFileCopy(
		filepath.Join(cr.ManifestsRoot, operatorPatchBaseFolder, "templates/keys/keys_configs/gomplate.yaml"),
		filepath.Join(cr.ManifestsRoot, operatorPatchBaseFolder, "configs/keys/gomplate.yaml"), 0777, nil); err != nil {
		return err
	}
	return nil
}

func writeKeysEjwksJsonFile(cr *config.CRConfig, services []*serviceT) error {
	templateFilePath := filepath.Join(cr.ManifestsRoot, operatorPatchBaseFolder, "templates/keys/keys_configs/ejwks.json.tmpl")
	destinationFilePath := filepath.Join(cr.ManifestsRoot, operatorPatchBaseFolder, "configs/keys/ejwks.json")
	var byteBuffer bytes.Buffer
	if template, err := template.New(path.Base(templateFilePath)).ParseFiles(templateFilePath); err != nil {
		return err
	} else if err := template.Execute(&byteBuffer, services); err != nil {
		return err
	} else if file, err := os.Create(destinationFilePath); err != nil {
		return err
	} else if _, err := ejson.Encrypt(&byteBuffer, file); err != nil {
		return err
	}
	return nil
}

func writeKeysSelectivePatchYamlFile(cr *config.CRConfig, services []*serviceT) error {
	templateFilePath := filepath.Join(cr.ManifestsRoot, operatorPatchBaseFolder, "templates/keys/keys_configs/selectivepatch.yaml.tmpl")
	destinationFilePath := filepath.Join(cr.ManifestsRoot, operatorPatchBaseFolder, "configs/keys/selectivepatch.yaml")
	if template, err := template.New(path.Base(templateFilePath)).ParseFiles(templateFilePath); err != nil {
		return err
	} else if file, err := os.Create(destinationFilePath); err != nil {
		return err
	} else if err := template.Execute(file, services); err != nil {
		return err
	}
	return nil
}

func smallFileCopy(srcPath string, dstPath string, perm os.FileMode, transform func([]byte) ([]byte, error)) error {
	bytes, err := ioutil.ReadFile(srcPath)
	if err != nil {
		return err
	}
	if transform != nil {
		bytes, err = transform(bytes)
		if err != nil {
			return err
		}
	}
	err = ioutil.WriteFile(dstPath, bytes, perm)
	if err != nil {
		return err
	}
	return nil
}
