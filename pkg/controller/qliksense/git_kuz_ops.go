package qliksense

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/mitchellh/go-homedir"
	kapis_config "github.com/qlik-oss/k-apis/pkg/config"
	kapis_cr "github.com/qlik-oss/k-apis/pkg/cr"
	_ "github.com/qlik-oss/k-apis/pkg/git"
	kapis_git "github.com/qlik-oss/k-apis/pkg/git"
	qlikv1 "github.com/qlik-oss/qliksense-operator/pkg/apis/qlik/v1"
	"io"
	"io/ioutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/kustomize/api/filesys"
	"sigs.k8s.io/kustomize/api/konfig"
	"sigs.k8s.io/kustomize/api/krusty"
	"sigs.k8s.io/kustomize/api/types"
)

const (
	MANIFEST_ON_DISK_ROOT = "/tmp/qlik-k8s"
	Q_INIT_CRD_PATH       = "manifests/base/manifests/qliksense-init"
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
	if manifestRoot, err := cloneGitRepo(qs.GetName(), qs.GetVersion(), &qs.Spec.Git); err != nil {
		return err
	} else {
		qs.Spec.ManifestsRoot = manifestRoot
		qi.ManifestRootMap[qs.GetName()] = manifestRoot
	}

	return nil
}

func (qi *QliksenseInstances) RemoveFromQliksenseInstances(crName string) error {

	if err := os.RemoveAll(qi.ManifestRootMap[crName]); err != nil {
		return err
	}
	delete(qi.ManifestRootMap, crName)
	delete(qi.InstanceMap, crName)
	return nil
}

func (qi *QliksenseInstances) GetCRSpec(crName string) *kapis_config.CRSpec {
	q := qi.InstanceMap[crName]
	return &q.Spec
}

func cloneGitRepo(crName, ref string, gRepo *kapis_config.Repo) (string, error) {
	destDir := filepath.Join(MANIFEST_ON_DISK_ROOT, crName, ref)
	if err := os.MkdirAll(destDir, os.ModePerm); err != nil {
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

func (qi *QliksenseInstances) installQliksene(crName string) error {
	crspec := qi.GetCRSpec(crName)
	userHomeDir, err := homedir.Dir()
	if err != nil {
		fmt.Printf(`error fetching user's home directory: %v\n`, err)
		return err
	}
	// generate patches
	kapis_cr.GeneratePatches(crspec, path.Join(userHomeDir, ".kube", "config"))
	qInitByte, err := executeKustomizeBuild(filepath.Join(crspec.GetManifestsRoot(), crspec.GetProfileDir()))
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
		fmt.Println("cannot create file ", err)
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
		fmt.Printf("kubectl apply failed with %s\n", err)
		fmt.Println("temp CRD file: " + tempYaml.Name())
		return err
	}
	os.Remove(tempYaml.Name())
	return nil
}
