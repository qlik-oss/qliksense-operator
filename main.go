package main

import (
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"

	"github.com/Shopify/ejson"
	"github.com/qlik-oss/qliksense-operator/pkg/config"
	"github.com/qlik-oss/qliksense-operator/pkg/qust"
	"github.com/qlik-oss/qliksense-operator/pkg/state"
)

const (
	defaultEjsonKeydir  = "/opt/ejson/keys"
	kubeConfigPath      = "/root/.kube/config"
	backupConfigMapName = "qliksense-operator-state-backup"
)

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
		generateKeys(cr, defaultEjsonKeydir)
		backupKeys(cr, defaultEjsonKeydir)
	} else {
		restoreKeys(cr, defaultEjsonKeydir)
	}
}

func generateKeys(cr *config.CRConfig, defaultKeyDir string) {
	log.Println("rotating all keys")
	keyDir := getEjsonKeyDir(defaultKeyDir)
	if ejsonPublicKey, ejsonPrivateKey, err := ejson.GenerateKeypair(); err != nil {
		log.Printf("error generating an ejson key pair: %v\n", err)
	} else if err := qust.GenerateKeys(cr, ejsonPublicKey); err != nil {
		log.Printf("error generating application keys: %v\n", err)
	} else if err := os.MkdirAll(keyDir, os.ModePerm); err != nil {
		log.Printf("error makeing sure private key storage directory: %v exists, error: %v\n", keyDir, err)
	} else if err := ioutil.WriteFile(path.Join(keyDir, ejsonPublicKey), []byte(ejsonPrivateKey), os.ModePerm); err != nil {
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

func backupKeys(cr *config.CRConfig, defaultKeyDir string) {
	log.Println("backing up keys into the cluster")
	if err := state.Backup(kubeConfigPath, backupConfigMapName, cr.NameSpace, []state.BackupDir{
		{ConfigmapKey: "operator-keys", Directory: filepath.Join(cr.ManifestsRoot, ".operator/keys")},
		{ConfigmapKey: "ejson-keys", Directory: getEjsonKeyDir(defaultKeyDir)},
	}); err != nil {
		log.Printf("error backing up keys data to the cluster, error: %v\n", err)
	}
}

func restoreKeys(cr *config.CRConfig, defaultKeyDir string) {
	log.Println("restoring keys from the cluster")
	if err := state.Restore(kubeConfigPath, backupConfigMapName, cr.NameSpace, []state.BackupDir{
		{ConfigmapKey: "operator-keys", Directory: filepath.Join(cr.ManifestsRoot, ".operator/keys")},
		{ConfigmapKey: "ejson-keys", Directory: getEjsonKeyDir(defaultKeyDir)},
	}); err != nil {
		log.Printf("error restoring keys data from the cluster, error: %v\n", err)
	}
}
