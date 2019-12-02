package main

import (
	"log"

	"github.com/qlik-oss/qliksense-operator/pkg/config"
	"github.com/qlik-oss/qliksense-operator/pkg/qust"
)

func main() {
	log.Println("running qliksense-operator .... ")
	cr, err := config.ReadCRConfigFromEnvYaml()
	if err != nil {
		log.Panic("Something wrong in CR ", err)
	}
	if cr.ManifestsRoot != "" {
		log.Println("manifests are in local file system")
		// no repo opeations needed
		processInLocalFileSystem(cr)
	}
	log.Println("exiting qliksense-operator")
}

func processInLocalFileSystem(cr *config.CRConfig) {
	// Process cr.configs
	qust.ProcessCrConfigs(cr)
	// Process cr.secrets
	qust.ProcessCrSecrets(cr)

	// process cr.storageClassName
	qust.ProcessStorageClassName(cr)

	if cr.GenerateKeys {
		err := qust.GenerateKeys(cr)
		if err != nil {
			log.Printf("error generating keys: %v\n", err)
		}
	}
}
