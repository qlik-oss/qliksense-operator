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

	"sigs.k8s.io/kustomize/api/k8sdeps/kunstruct"
	"sigs.k8s.io/kustomize/api/resmap"
	"sigs.k8s.io/kustomize/api/resource"

	"github.com/mholt/archiver/v3"

	"github.com/qlik-oss/k-apis/pkg/git"
)

func Test_startKuzHttpServer_kuzHandler(t *testing.T) {
	if os.Getenv("EXECUTE_K8S_TESTS") != "true" {
		t.SkipNow()
	}

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
