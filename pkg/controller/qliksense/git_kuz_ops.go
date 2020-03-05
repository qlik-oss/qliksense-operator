package qliksense

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/go-logr/logr"
	"github.com/google/uuid"
	kapis_config "github.com/qlik-oss/k-apis/pkg/config"
	kapis_cr "github.com/qlik-oss/k-apis/pkg/cr"
	_ "github.com/qlik-oss/k-apis/pkg/git"
	kapis_git "github.com/qlik-oss/k-apis/pkg/git"
	qlikv1 "github.com/qlik-oss/qliksense-operator/pkg/apis/qlik/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/kustomize/api/filesys"
	"sigs.k8s.io/kustomize/api/konfig"
	"sigs.k8s.io/kustomize/api/krusty"
	"sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/yaml"
)

const (
	Q_INIT_CRD_PATH = "manifests/base/manifests/qliksense-init"
)

type QliksenseInstances struct {
	InstanceMap     map[string]*qlikv1.Qliksense
	ManifestRootMap map[string]string
}

func NewQIs() *QliksenseInstances {
	return &QliksenseInstances{
		InstanceMap:     make(map[string]*qlikv1.Qliksense),
		ManifestRootMap: make(map[string]string),
	}
}
func (qi *QliksenseInstances) AddToQliksenseInstances(qs *qlikv1.Qliksense) error {
	qi.InstanceMap[qs.GetName()] = qs
	if manifestRoot, err := cloneGitRepo(qs.GetName(), qs.GetVersion(), qs.Spec.Git); err != nil {
		return err
	} else {
		qs.Spec.ManifestsRoot = manifestRoot
		qi.ManifestRootMap[qs.GetName()] = manifestRoot
	}

	return nil
}

func getKuzLogger() logr.Logger {
	return log.WithValues("activities", "performing-git-kuz-ops")
}
func (qi *QliksenseInstances) RemoveFromQliksenseInstances(crName string) error {

	if err := os.RemoveAll(qi.ManifestRootMap[crName]); err != nil {
		return err
	}
	delete(qi.ManifestRootMap, crName)
	delete(qi.InstanceMap, crName)
	return nil
}

func (qi *QliksenseInstances) GetCRSpec(crName string) *qlikv1.CombinedSpec {
	q := qi.InstanceMap[crName]
	return q.Spec
}

// clone the git repo and checking out a branch out of it
func cloneGitRepo(crName, ref string, gRepo *kapis_config.Repo) (string, error) {
	crRoot, err := getCrRootDir(crName)
	if err != nil {
		getKuzLogger().Error(err, "cannot get root cr root directory")
		return "", err
	}
	destDir := filepath.Join(crRoot, ref)
	if err := os.MkdirAll(destDir, os.ModePerm); err != nil {
		getKuzLogger().Error(err, "cannot create directory")
		return "", err
	}
	if b, err := IsDirEmpty(destDir); err != nil {
		//previously cloned
		return destDir, nil
	} else if b {
		if repo, err := kapis_git.CloneRepository(destDir, gRepo.Repository, nil); err != nil {
			return "", err
		} else if err = kapis_git.Checkout(repo, ref, fmt.Sprintf("%v-by-operator-%v", ref, uuid.New().String()), nil); err != nil {
			return "", err
		}
		return destDir, nil
	}
	return destDir, nil
}

func getCrRootDir(crName string) (string, error) {
	dirName, err := ioutil.TempDir("", "")
	if err != nil {
		return "", err
	}
	destDir := filepath.Join(dirName, "qlik-k8s", crName)
	if err := os.MkdirAll(destDir, os.ModePerm); err != nil {
		return "", err
	} else {
		return destDir, nil
	}
}

// IsInstalled verify if qliksense is installed based on engine resources availability
func (qi *QliksenseInstances) IsInstalled(crName string) bool {
	q := qi.InstanceMap[crName]
	if q == nil {
		return false
	}
	// Get a config to talk to the apiserver
	cfg, err := config.GetConfig()
	if err != nil {
		return false
	}

	dynamicClient, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return false
	}

	engineRes := schema.GroupVersionResource{Group: "qixmanager.qlik.com", Version: "v1", Resource: "engines"}

	list, err := dynamicClient.Resource(engineRes).Namespace(q.Namespace).List(metav1.ListOptions{})
	if err != nil {
		return false
	}

	for _, d := range list.Items {
		rls, _, err := unstructured.NestedString(d.Object, "metadata", "labels", searchingLabel)
		if err != nil {
			return false
		}
		if rls == q.Name {
			return true
		}
	}
	return false
}

func IsDirEmpty(name string) (bool, error) {
	f, err := os.Open(name)
	if err != nil {
		return false, err
	}
	defer f.Close()

	_, err = f.Readdirnames(1) // Or f.Readdir(1)
	if err == io.EOF {
		return true, nil
	}
	return false, err // Either not empty or error, suits both cases
}

func executeKustomizeBuild(directory string) ([]byte, error) {
	fSys := filesys.MakeFsOnDisk()
	options := &krusty.Options{
		DoLegacyResourceSort: false,
		LoadRestrictions:     types.LoadRestrictionsNone,
		DoPrune:              false,
		PluginConfig:         konfig.DisabledPluginConfig(),
	}
	k := krusty.MakeKustomizer(fSys, options)
	resMap, err := k.Run(directory)
	if err != nil {
		return nil, err
	}
	return resMap.AsYaml()
}

func (qi *QliksenseInstances) installQliksene(qse *qlikv1.Qliksense) error {
	mfestRoot := qi.ManifestRootMap[qse.GetName()]
	qse.Spec.ManifestsRoot = mfestRoot
	kcr := convertToKApiCr(qse)
	// generate patches
	// empty string should use in-cluster config
	dirName, _ := ioutil.TempDir("", "")
	if err := os.Setenv("EJSON_KEYDIR", dirName); err != nil {
		getKuzLogger().Error(err, "cannot set env for EJSON_KEYDIR")
	}
	getKuzLogger().Info("generating kustomize patches by k-api")
	kapis_cr.GeneratePatches(kcr, "")
	getKuzLogger().Info("executing kustomize build in folder " + filepath.Join(kcr.Spec.GetManifestsRoot(), kcr.Spec.GetProfileDir()))
	qInitByte, err := executeKustomizeBuild(filepath.Join(kcr.Spec.GetManifestsRoot(), kcr.Spec.GetProfileDir()))
	if err != nil {
		return err
	}

	if err := KubectlApply(string(qInitByte)); err != nil {
		return err
	}

	return nil
}

func KubectlApply(manifests string) error {
	return kubectlOperation(manifests, "apply")
}

func KubectlDelete(manifests string) error {
	return kubectlOperation(manifests, "delete")
}

func kubectlOperation(manifests string, oprName string) error {
	tempYaml, err := ioutil.TempFile("", "")
	if err != nil {
		getKuzLogger().Error(err, "cannot create file ")
		return err
	}
	tempYaml.WriteString(manifests)

	var cmd *exec.Cmd
	if oprName == "apply" {
		cmd = exec.Command("kubectl", oprName, "-f", tempYaml.Name(), "--validate=false")
	} else {
		cmd = exec.Command("kubectl", oprName, "-f", tempYaml.Name())
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		getKuzLogger().Error(err, "kubectl apply failed , file"+tempYaml.Name())
		return err
	}
	os.Remove(tempYaml.Name())
	return nil
}

func convertToKApiCr(qse *qlikv1.Qliksense) *kapis_config.KApiCr {
	return &kapis_config.KApiCr{
		TypeMeta:   qse.TypeMeta,
		ObjectMeta: qse.ObjectMeta,
		Spec:       qse.Spec.CRSpec,
	}
}

func K8sToYaml(k8sObj interface{}) ([]byte, error) {
	k8sSecretYamlMap := map[string]interface{}{}
	if k8sSecretYamlBytes, err := yaml.Marshal(k8sObj); err != nil {
		return nil, err
	} else if err := yaml.Unmarshal(k8sSecretYamlBytes, &k8sSecretYamlMap); err != nil {
		return nil, err
	} else {
		delete(k8sSecretYamlMap["metadata"].(map[string]interface{}), "creationTimestamp")
		return yaml.Marshal(k8sSecretYamlMap)
	}
}
