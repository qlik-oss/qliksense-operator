package main

import (
	"github.com/qlik-oss/qliksense-operator/pkg/config"
	"github.com/qlik-oss/qliksense-operator/pkg/qust"
	"log"
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
}
