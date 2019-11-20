package qust

import (
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"

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
		service.PrivateKey = strings.Replace(service.PrivateKey, "\n", "\\n", -1)

		if err := setupServiceSecretsDirectory(cr, &service); err != nil {
			return err
		}
		if err := writeServiceEpriviteKeyJsonFile(cr, &service); err != nil {
			return err
		}
		if err := writeServiceSelectivePatchYamlFile(cr, &service); err != nil {
			return err
		}
	}
	return nil
}

func initServiceList(cr *config.CRConfig) ([]serviceT, error) {
	serviceListYamlPath := filepath.Join(cr.ManifestsRoot, operatorPatchBaseFolder, "templates/keys/services.yaml")
	yamlBytes, err := ioutil.ReadFile(serviceListYamlPath)
	if err != nil {
		return nil, err
	}

	var serviceList []serviceT
	err = yaml.Unmarshal(yamlBytes, &serviceList)
	if err != nil {
		return nil, err
	}

	return serviceList, nil
}

func setupServiceSecretsDirectory(cr *config.CRConfig, service *serviceT) error {
	serviceSecretsDirPath := filepath.Join(cr.ManifestsRoot, operatorPatchBaseFolder, "secrets", service.Name)
	if err := os.RemoveAll(serviceSecretsDirPath); err != nil {
		return err
	}
	if err := os.Mkdir(serviceSecretsDirPath, 0777); err != nil {
		return err
	}

	tmplKustomizationFilePath := filepath.Join(cr.ManifestsRoot, operatorPatchBaseFolder, "templates/keys/service_secrets/kustomization.yaml")
	kustomizationFileBytes, err := ioutil.ReadFile(tmplKustomizationFilePath)
	if err != nil {
		return err
	}

	serviceKustomizationFilePath := filepath.Join(cr.ManifestsRoot, operatorPatchBaseFolder, "secrets", service.Name, "kustomization.yaml")
	err = ioutil.WriteFile(serviceKustomizationFilePath, kustomizationFileBytes, 0777)
	if err != nil {
		return err
	}
	return nil
}

func writeServiceEpriviteKeyJsonFile(cr *config.CRConfig, service *serviceT) error {
	templateFilePath := filepath.Join(cr.ManifestsRoot, operatorPatchBaseFolder, "templates/keys/service_secrets/eprivate_key.json.tmpl")
	destinationFilePath := filepath.Join(cr.ManifestsRoot, operatorPatchBaseFolder, "secrets", service.Name, "eprivate_key.json")
	if template, err := template.New(path.Base(templateFilePath)).ParseFiles(templateFilePath); err != nil {
		return err
	} else if file, err := os.Create(destinationFilePath); err != nil {
		return err
	} else if err := template.Execute(file, service); err != nil {
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
