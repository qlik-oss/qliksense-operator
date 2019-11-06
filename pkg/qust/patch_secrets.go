package qust

import (
	"github.com/qlik-oss/qliksense-operator/pkg/config"
	"gopkg.in/yaml.v2"
	"log"
	"os"
	"path/filepath"
	"sigs.k8s.io/kustomize/api/types"
)

func ProcessCrSecrets(cr *config.CRConfig) {
	pm := createSupperSecretSelectivePatch(cr.Secrets)
	baseSecretDir := filepath.Join(cr.ManifestsRoot, operatorPatchBaseFolder, "secrets")
	for svc, sps := range pm {
		fpath := filepath.Join(baseSecretDir, svc+".yaml")
		fileHand, _ := os.Create(fpath)
		YamlToWriter(fileHand, sps)
		err := addResourceToKustomization(svc+".yaml", filepath.Join(baseSecretDir, "kustomization.yaml"))
		if err != nil {
			log.Println("Cannot process secrets", err)
		}
	}
}

// create a selectivepatch map for each service for a secretKey
func createSupperSecretSelectivePatch(sec []config.Secret) map[string]*config.SelectivePatch {
	spMap := make(map[string]*config.SelectivePatch)
	for _, se := range sec {
		for svc, v := range se.Values {
			p := getSecretPatchBody(se.SecretKey, svc, v)
			sp := getSuperSecretSPTemplate(svc)
			sp.Patches = []types.Patch{p}
			mergeSelectivePatches(sp, spMap[svc])
			spMap[svc] = sp

		}
	}
	return spMap
}

// create a patch section to be added to the selective patch
func getSecretPatchBody(secretKey, svc, value string) types.Patch {
	ph := getSuperSecretTemplate(svc)
	ph.StringData = map[string]string{
		secretKey: value,
	}
	// ph := `
	// 	apiVersion: qlik.com/v1
	// 	kind: SuperSecret
	// 	metadata:
	// 		name: ` + svc + `-secrets
	// 	data:
	// 		` + dataKey + `: ` + value

	// target:
	//   kind: SuperSecret
	//   labelSelector: "app=" + svc,
	phb, _ := yaml.Marshal(ph)
	p1 := types.Patch{
		Patch:  string(phb),
		Target: getSelector("SuperSecret", svc),
	}
	return p1
}

// a SelectivePatch object with service name in it
func getSuperSecretSPTemplate(svc string) *config.SelectivePatch {
	su := &config.SelectivePatch{
		ApiVersion: "qlik.com/v1",
		Kind:       "SelectivePatch",
		Metadata: map[string]string{
			"name": svc + "-operator-secrets",
		},
		Enabled: true,
	}
	return su
}

func getSuperSecretTemplate(svc string) *config.SupperSecret {
	return &config.SupperSecret{
		ApiVersion: "qlik.com/v1",
		Kind:       "SuperSecret",
		Metadata: map[string]string{
			"name": svc + "-secrets",
		},
	}
}
