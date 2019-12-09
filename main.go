package main

import (
	"io/ioutil"
	"log"
	"os"
	"path"

	"github.com/Shopify/ejson"
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
		log.Printf("manifests are in local file system in %v\n", cr.ManifestsRoot)
		// no repo opeations needed
		processInLocalFileSystem(cr)
	}
	log.Println("exiting qliksense-operator")
}

func processInLocalFileSystem(cr *config.CRConfig) {

	// process cr.storageClassName
	if cr.StorageClassName != "" {
		qust.ProcessStorageClassName(cr)
		// added to the configs so that down the road it is being processed
		c := config.Config{
			DataKey: "storageClassName",
			Values: map[string]string{
				"qliksense": cr.StorageClassName,
			},
		}
		cr.Configs = append(cr.Configs, c)
	}
	// process cr.Namespace
	qust.ProcessNamespace(cr)

	// Process cr.configs
	qust.ProcessCrConfigs(cr)
	// Process cr.secrets
	qust.ProcessCrSecrets(cr)

	if cr.RotateKeys == "yes" {
		log.Println("rotating all keys")
		generateKeys(cr, defaultEjsonKeydir)
	}
}

func generateKeys(cr *config.CRConfig, defaultKeyDir string) {
	keyDir := getEjsonKeyDir(defaultKeyDir)
	if ejsonPublicKey, ejsonPrivateKey, err := ejson.GenerateKeypair(); err != nil {
		log.Printf("error generating an ejson key pair: %v\n", err)
	} else if err := qust.GenerateKeys(cr, ejsonPublicKey); err != nil {
		log.Printf("error generating application keys: %v\n", err)
	} else if err := os.MkdirAll(keyDir, 0777); err != nil {
		log.Printf("error makeing sure private key storage directory: %v exists, error: %v\n", keyDir, err)
	} else if err := ioutil.WriteFile(path.Join(keyDir, ejsonPublicKey), []byte(ejsonPrivateKey), 0777); err != nil {
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
