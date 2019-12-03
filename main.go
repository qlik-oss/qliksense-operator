package main

import (
	"github.com/Shopify/ejson"
	"io/ioutil"
	"log"
	"os"
	"path"

	"github.com/qlik-oss/qliksense-operator/pkg/config"
	"github.com/qlik-oss/qliksense-operator/pkg/qust"
)

const defaultEjsonKeydir = "/opt/ejson/keys"

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

	// process cr.storageClassName
	if cr.StorageClassName != "" {
		qust.ProcessStorageClassName(cr)
		c := config.Config{
			DataKey: "storageClassName",
			Values: map[string]string{
				"qliksense": cr.StorageClassName,
			},
		}
		cr.Configs = append(cr.Configs, c)
	}

	// Process cr.configs
	qust.ProcessCrConfigs(cr)
	// Process cr.secrets
	qust.ProcessCrSecrets(cr)

	// process cr.storageClassName
	qust.ProcessStorageClassName(cr)

	log.Printf("cr.RotateKeys value is: %v\n", cr.RotateKeys)
	if cr.RotateKeys == "yes" {
		generateKeys(cr, defaultEjsonKeydir)
	}
}

func generateKeys(cr *config.CRConfig, defaultKeyDir string) {
	if ejsonPublicKey, ejsonPrivateKey, err := ejson.GenerateKeypair(); err != nil {
		log.Printf("error generating an ejson key pair: %v\n", err)
	} else if err := qust.GenerateKeys(cr, ejsonPublicKey); err != nil {
		log.Printf("error generating application keys: %v\n", err)
	} else if err := ioutil.WriteFile(path.Join(getEjsonKeyDir(defaultKeyDir), ejsonPublicKey), []byte(ejsonPrivateKey), 0777); err != nil {
		log.Printf("error storing ejson private key: %v\n", err)
	}
}

func getEjsonKeyDir(defaultKeyDir string) string {
	ejsonKeyDir := os.Getenv("EJSON_KEYDIR")
	if ejsonKeyDir == "" {
		ejsonKeyDir = defaultKeyDir
	}
	return ejsonKeyDir
}
