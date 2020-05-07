package qliksense

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	kapis_config "github.com/qlik-oss/k-apis/pkg/config"
	machine_yaml "k8s.io/apimachinery/pkg/util/yaml"

	"sigs.k8s.io/kustomize/api/k8sdeps/kunstruct"
	"sigs.k8s.io/kustomize/api/resmap"
	"sigs.k8s.io/kustomize/api/resource"

	"github.com/mholt/archiver/v3"

	"github.com/qlik-oss/k-apis/pkg/git"
)

func Test_startKuzHttpServer_kuzHandler(t *testing.T) {
	srv := startKuzHttpServer("localhost", 8080)
	defer srv.Close()

	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	configPath := filepath.Join(tmpDir, "repo")
	if r, err := git.CloneRepository(configPath, "https://github.com/qlik-oss/qliksense-k8s", nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	} else if err := git.Checkout(r, "v0.0.8", "", nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var archiveSources []string
	if fileInfos, err := ioutil.ReadDir(configPath); err != nil {
		t.Fatalf("unexpected error: %v", err)
	} else {
		for _, fileInfo := range fileInfos {
			archiveSources = append(archiveSources, filepath.Join(configPath, fileInfo.Name()))
		}
	}
	if err := archiver.NewTarGz().Archive(archiveSources, filepath.Join(tmpDir, "repo.tgz")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	configTgzBytes, err := ioutil.ReadFile(filepath.Join(tmpDir, "repo.tgz"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cr := `
apiVersion: qlik.com/v1
kind: Qliksense
metadata:
  name: qlik-default
  labels:
    version: v0.0.8
spec:
  profile: docker-desktop
  git:
    repository: https://github.com/qlik-oss/qliksense-k8s
    accessToken: ""
    userName: ""
  opsRunner:
    enabled: "yes"
    schedule: "*/2 * * * *"
    watchBranch: master
    image: qlik/qliksense-repo-watcher:latest
  secrets:
    qliksense:
      - name: mongoDbUri
        value: mongodb://qlik-default-mongodb:27017/qliksense?ssl=false
  configs:
    qliksense:
      - name: acceptEULA
        value: "yes"
  rotateKeys: "yes"
`
	requestMap := map[string]string{
		"cr":     base64.StdEncoding.EncodeToString([]byte(cr)),
		"config": base64.StdEncoding.EncodeToString(configTgzBytes),
	}
	requestMapJsonBytes, err := json.Marshal(requestMap)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	resp, err := http.Post("http://localhost:8080/kuz", "Content-Type: application/json", bytes.NewReader(requestMapJsonBytes))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	respBodyMap := make(map[string]string)
	resmapFactory := resmap.NewFactory(resource.NewFactory(kunstruct.NewKunstructuredFactoryImpl()), nil)

	if respBodyJsonBytes, err := ioutil.ReadAll(resp.Body); err != nil {
		t.Fatalf("unexpected error: %v", err)
	} else if err := json.Unmarshal(respBodyJsonBytes, &respBodyMap); err != nil {
		t.Fatalf("unexpected error unmarshalling response to json map, response: %v, error: %v", string(respBodyJsonBytes), err)
	} else if manifestsTzpBytes, err := base64.StdEncoding.DecodeString(respBodyMap["manifests"]); err != nil {
		t.Fatalf("unexpected error: %v", err)
	} else if err := ioutil.WriteFile(filepath.Join(tmpDir, "manifests.tgz"), manifestsTzpBytes, os.ModePerm); err != nil {
		t.Fatalf("unexpected error: %v", err)
	} else if err := os.MkdirAll(filepath.Join(tmpDir, "manifests"), os.ModePerm); err != nil {
		t.Fatalf("unexpected error: %v", err)
	} else if err := archiver.NewTarGz().Unarchive(filepath.Join(tmpDir, "manifests.tgz"), filepath.Join(tmpDir, "manifests")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	} else if manifestBytes, err := ioutil.ReadFile(filepath.Join(tmpDir, "manifests", "manifest.yaml")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	} else if _, err := resmapFactory.NewResMapFromBytes(manifestBytes); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func Test_createTarGz(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	if tarZipBytes, err := createTarGz("foo", []byte("foobar")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	} else if err := ioutil.WriteFile(filepath.Join(tmpDir, "test.tgz"), tarZipBytes, os.ModePerm); err != nil {
		t.Fatalf("unexpected error: %v", err)
	} else if err := os.MkdirAll(filepath.Join(tmpDir, "test"), os.ModePerm); err != nil {
		t.Fatalf("unexpected error: %v", err)
	} else if err := archiver.NewTarGz().Unarchive(filepath.Join(tmpDir, "test.tgz"), filepath.Join(tmpDir, "test")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	} else if fooBytes, err := ioutil.ReadFile(filepath.Join(tmpDir, "test", "foo")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	} else if string(fooBytes) != "foobar" {
		t.Fatalf("expected: %v, but got: %v", "foobar", string(fooBytes))
	}
}

func Test_CR_decode(t *testing.T) {
	type testCaseT struct {
		name   string
		cr     string
		verify func(t *testing.T, kcr *kapis_config.KApiCr)
	}
	var testCases = []testCaseT{
		{
			name: "expect yes to true conversion",
			cr:   "apiVersion: qlik.com/v1\nkind: Qliksense\nmetadata:\n  annotations:\n    kubectl.kubernetes.io/last-applied-configuration: |\n      {\"apiVersion\":\"qlik.com/v1\",\"kind\":\"Qliksense\",\"metadata\":{\"annotations\":{},\"labels\":{\"version\":\"v0.0.8\"},\"name\":\"qlik-default\",\"namespace\":\"default\"},\"spec\":{\"configs\":{\"qliksense\":[{\"name\":\"acceptEULA\",\"value\":\"yes\"}]},\"git\":{\"accessToken\":\"\",\"repository\":\"\",\"userName\":\"\"},\"opsRunner\":{\"enabled\":\"yes\",\"image\":\"qlik/qliksense-repo-watcher:andriy-burnt-runner-4\",\"schedule\":\"\",\"watchBranch\":\"\"},\"profile\":\"docker-desktop\",\"rotateKeys\":\"yes\",\"secrets\":{\"qliksense\":[{\"name\":\"mongoDbUri\",\"value\":\"mongodb://qlik-default-mongodb:27017/qliksense?ssl=false\"}]}}}\n  generation: 1\n  labels:\n    version: v0.0.8\n  name: qlik-default\n  namespace: default\n  resourceVersion: \"260103\"\n  selfLink: /apis/qlik.com/v1/namespaces/default/qliksenses/qlik-default/status\n  uid: 028c7a19-6b4d-4971-8e1b-7a39f617dcbc\nspec:\n  configs:\n    qliksense:\n    - name: acceptEULA\n      value: \"yes\"\n  git:\n    repository: \"\"\n  opsRunner:\n    enabled: \"yes\"\n    image: qlik/qliksense-repo-watcher:andriy-burnt-runner-4\n  profile: docker-desktop\n  rotateKeys: yes\n  secrets:\n    qliksense:\n    - name: mongoDbUri\n      value: mongodb://qlik-default-mongodb:27017/qliksense?ssl=false\nstatus:\n  conditions:\n  - lastTransitionTime: \"2020-05-06T06:15:17Z\"\n    status: Valid\n    type: CliMode\n  - lastTransitionTime: \"2020-05-06T06:15:17Z\"\n    status: Valid\n    type: Initialized\n",
			verify: func(t *testing.T, kcr *kapis_config.KApiCr) {
				if kcr.Spec.RotateKeys != "true" {
					t.Fatalf("expected rotateKeys to be true, but it was: %v", kcr.Spec.RotateKeys)
				}
			},
		},
		{
			name: "expect no to false conversion",
			cr:   "apiVersion: qlik.com/v1\nkind: Qliksense\nmetadata:\n  annotations:\n    kubectl.kubernetes.io/last-applied-configuration: |\n      {\"apiVersion\":\"qlik.com/v1\",\"kind\":\"Qliksense\",\"metadata\":{\"annotations\":{},\"labels\":{\"version\":\"v0.0.8\"},\"name\":\"qlik-default\",\"namespace\":\"default\"},\"spec\":{\"configs\":{\"qliksense\":[{\"name\":\"acceptEULA\",\"value\":\"yes\"}]},\"git\":{\"accessToken\":\"\",\"repository\":\"\",\"userName\":\"\"},\"opsRunner\":{\"enabled\":\"yes\",\"image\":\"qlik/qliksense-repo-watcher:andriy-burnt-runner-4\",\"schedule\":\"\",\"watchBranch\":\"\"},\"profile\":\"docker-desktop\",\"rotateKeys\":\"yes\",\"secrets\":{\"qliksense\":[{\"name\":\"mongoDbUri\",\"value\":\"mongodb://qlik-default-mongodb:27017/qliksense?ssl=false\"}]}}}\n  generation: 1\n  labels:\n    version: v0.0.8\n  name: qlik-default\n  namespace: default\n  resourceVersion: \"260103\"\n  selfLink: /apis/qlik.com/v1/namespaces/default/qliksenses/qlik-default/status\n  uid: 028c7a19-6b4d-4971-8e1b-7a39f617dcbc\nspec:\n  configs:\n    qliksense:\n    - name: acceptEULA\n      value: \"yes\"\n  git:\n    repository: \"\"\n  opsRunner:\n    enabled: \"yes\"\n    image: qlik/qliksense-repo-watcher:andriy-burnt-runner-4\n  profile: docker-desktop\n  rotateKeys: no\n  secrets:\n    qliksense:\n    - name: mongoDbUri\n      value: mongodb://qlik-default-mongodb:27017/qliksense?ssl=false\nstatus:\n  conditions:\n  - lastTransitionTime: \"2020-05-06T06:15:17Z\"\n    status: Valid\n    type: CliMode\n  - lastTransitionTime: \"2020-05-06T06:15:17Z\"\n    status: Valid\n    type: Initialized\n",
			verify: func(t *testing.T, kcr *kapis_config.KApiCr) {
				if kcr.Spec.RotateKeys != "false" {
					t.Fatalf("expected rotateKeys to be false, but it was: %v", kcr.Spec.RotateKeys)
				}
			},
		},
		{
			name: `expect "yes" to yes conversion`,
			cr:   "apiVersion: qlik.com/v1\nkind: Qliksense\nmetadata:\n  annotations:\n    kubectl.kubernetes.io/last-applied-configuration: |\n      {\"apiVersion\":\"qlik.com/v1\",\"kind\":\"Qliksense\",\"metadata\":{\"annotations\":{},\"labels\":{\"version\":\"v0.0.8\"},\"name\":\"qlik-default\",\"namespace\":\"default\"},\"spec\":{\"configs\":{\"qliksense\":[{\"name\":\"acceptEULA\",\"value\":\"yes\"}]},\"git\":{\"accessToken\":\"\",\"repository\":\"\",\"userName\":\"\"},\"opsRunner\":{\"enabled\":\"yes\",\"image\":\"qlik/qliksense-repo-watcher:andriy-burnt-runner-4\",\"schedule\":\"\",\"watchBranch\":\"\"},\"profile\":\"docker-desktop\",\"rotateKeys\":\"yes\",\"secrets\":{\"qliksense\":[{\"name\":\"mongoDbUri\",\"value\":\"mongodb://qlik-default-mongodb:27017/qliksense?ssl=false\"}]}}}\n  generation: 1\n  labels:\n    version: v0.0.8\n  name: qlik-default\n  namespace: default\n  resourceVersion: \"260103\"\n  selfLink: /apis/qlik.com/v1/namespaces/default/qliksenses/qlik-default/status\n  uid: 028c7a19-6b4d-4971-8e1b-7a39f617dcbc\nspec:\n  configs:\n    qliksense:\n    - name: acceptEULA\n      value: \"yes\"\n  git:\n    repository: \"\"\n  opsRunner:\n    enabled: \"yes\"\n    image: qlik/qliksense-repo-watcher:andriy-burnt-runner-4\n  profile: docker-desktop\n  rotateKeys: \"yes\"\n  secrets:\n    qliksense:\n    - name: mongoDbUri\n      value: mongodb://qlik-default-mongodb:27017/qliksense?ssl=false\nstatus:\n  conditions:\n  - lastTransitionTime: \"2020-05-06T06:15:17Z\"\n    status: Valid\n    type: CliMode\n  - lastTransitionTime: \"2020-05-06T06:15:17Z\"\n    status: Valid\n    type: Initialized\n",
			verify: func(t *testing.T, kcr *kapis_config.KApiCr) {
				if kcr.Spec.RotateKeys != "yes" {
					t.Fatalf("expected rotateKeys to be yes, but it was: %v", kcr.Spec.RotateKeys)
				}
			},
		},
		{
			name: `expect "no" to no conversion`,
			cr:   "apiVersion: qlik.com/v1\nkind: Qliksense\nmetadata:\n  annotations:\n    kubectl.kubernetes.io/last-applied-configuration: |\n      {\"apiVersion\":\"qlik.com/v1\",\"kind\":\"Qliksense\",\"metadata\":{\"annotations\":{},\"labels\":{\"version\":\"v0.0.8\"},\"name\":\"qlik-default\",\"namespace\":\"default\"},\"spec\":{\"configs\":{\"qliksense\":[{\"name\":\"acceptEULA\",\"value\":\"yes\"}]},\"git\":{\"accessToken\":\"\",\"repository\":\"\",\"userName\":\"\"},\"opsRunner\":{\"enabled\":\"yes\",\"image\":\"qlik/qliksense-repo-watcher:andriy-burnt-runner-4\",\"schedule\":\"\",\"watchBranch\":\"\"},\"profile\":\"docker-desktop\",\"rotateKeys\":\"yes\",\"secrets\":{\"qliksense\":[{\"name\":\"mongoDbUri\",\"value\":\"mongodb://qlik-default-mongodb:27017/qliksense?ssl=false\"}]}}}\n  generation: 1\n  labels:\n    version: v0.0.8\n  name: qlik-default\n  namespace: default\n  resourceVersion: \"260103\"\n  selfLink: /apis/qlik.com/v1/namespaces/default/qliksenses/qlik-default/status\n  uid: 028c7a19-6b4d-4971-8e1b-7a39f617dcbc\nspec:\n  configs:\n    qliksense:\n    - name: acceptEULA\n      value: \"yes\"\n  git:\n    repository: \"\"\n  opsRunner:\n    enabled: \"yes\"\n    image: qlik/qliksense-repo-watcher:andriy-burnt-runner-4\n  profile: docker-desktop\n  rotateKeys: \"no\"\n  secrets:\n    qliksense:\n    - name: mongoDbUri\n      value: mongodb://qlik-default-mongodb:27017/qliksense?ssl=false\nstatus:\n  conditions:\n  - lastTransitionTime: \"2020-05-06T06:15:17Z\"\n    status: Valid\n    type: CliMode\n  - lastTransitionTime: \"2020-05-06T06:15:17Z\"\n    status: Valid\n    type: Initialized\n",
			verify: func(t *testing.T, kcr *kapis_config.KApiCr) {
				if kcr.Spec.RotateKeys != "no" {
					t.Fatalf("expected rotateKeys to be no, but it was: %v", kcr.Spec.RotateKeys)
				}
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			var kcr kapis_config.KApiCr
			dec := machine_yaml.NewYAMLOrJSONDecoder(bytes.NewReader([]byte(testCase.cr)), 10000)
			if err := dec.Decode(&kcr); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			testCase.verify(t, &kcr)
		})
	}
}
