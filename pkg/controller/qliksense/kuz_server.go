package qliksense

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"time"

	kapis_config "github.com/qlik-oss/k-apis/pkg/config"

	"github.com/gorilla/mux"
	"github.com/mholt/archiver/v3"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	machine_yaml "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/rest"
	crclient "sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	kuzServerHost        = "0.0.0.0"
	kuzServicePort int32 = 7000
	kuzPortName          = "kuz-port"
	serverLog            = logf.Log.WithName("kuz_server")
)

func ConfigureAndStartKuzServer(ctx context.Context, cfg *rest.Config, _ string) (*http.Server, error) {
	if _, err := createKuzK8sService(ctx, cfg, kuzServicePort); err != nil {
		serverLog.Info("Could not create kustomize k8s Service", "error", err.Error())
		return nil, err
	}
	return startKuzHttpServer(kuzServerHost, kuzServicePort), nil
}

func startKuzHttpServer(host string, port int32) *http.Server {
	r := mux.NewRouter()
	r.HandleFunc("/health", healthCheckHandler).Methods("GET")
	r.HandleFunc("/kuz", kuzHandler).Methods("POST")

	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", host, port),
		WriteTimeout: time.Second * 60 * 10,
		ReadTimeout:  time.Second * 60 * 10,
		Handler:      r, // Pass our instance of gorilla/mux in.
	}
	serverLog.Info("starting kustomize HTTP server...")
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			serverLog.Info(fmt.Sprintf("kustomize HTTP server terminated with error: %v", err))
		}
	}()
	return srv
}

func healthCheckHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

func kuzHandler(w http.ResponseWriter, r *http.Request) {
	if crBytes, configTarZipBytes, err := parseKuzRequest(r); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	} else if configDir, configDirCleanup, err := stageKuzRequest(crBytes, configTarZipBytes); err != nil {
		http.Error(w, "", http.StatusInternalServerError)
		return
	} else {
		defer configDirCleanup()

		if manifestBytes, err := patchAndKustomizeConfig(crBytes, configDir); err != nil {
			serverLog.Error(err, "error patching/kustomizing config")
			http.Error(w, "", http.StatusInternalServerError)
			return
		} else if manifestTarZipBytes, err := createTarGz("manifest.yaml", manifestBytes); err != nil {
			serverLog.Error(err, "error creating a result tarball")
			http.Error(w, "", http.StatusInternalServerError)
			return
		} else {
			manifestTarZipBase64 := base64.StdEncoding.EncodeToString(manifestTarZipBytes)
			responseMap := map[string]string{"manifests": manifestTarZipBase64}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			if err := json.NewEncoder(w).Encode(responseMap); err != nil {
				serverLog.Error(err, "error marshalling result to json")
			}
		}
	}
}

func createTarGz(itemName string, itemBytes []byte) ([]byte, error) {
	buffer := &bytes.Buffer{}

	gzipWriter := gzip.NewWriter(buffer)
	defer gzipWriter.Close()

	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	header := &tar.Header{
		Name: itemName,
		Mode: 0666,
		Size: int64(len(itemBytes)),
	}
	if err := tarWriter.WriteHeader(header); err != nil {
		return nil, err
	} else if _, err := tarWriter.Write(itemBytes); err != nil {
		return nil, err
	} else if err := tarWriter.Close(); err != nil {
		return nil, err
	} else if err := gzipWriter.Close(); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func patchAndKustomizeConfig(cr []byte, configPath string) ([]byte, error) {
	var kcr kapis_config.KApiCr
	dec := machine_yaml.NewYAMLOrJSONDecoder(bytes.NewReader(cr), 10000)
	if err := dec.Decode(&kcr); err != nil {
		return nil, err
	} else {
		kcr.Spec.ManifestsRoot = configPath
		return PatchAndKustomize(&kcr)
	}
}

func parseKuzRequest(r *http.Request) (crBytes []byte, configTarZipBytes []byte, err error) {
	type kuzRequestObjectT struct {
		Cr     string `json:"cr,omitempty"`
		Config string `json:"config,omitempty"`
	}
	var kuzObject kuzRequestObjectT
	if err := json.NewDecoder(r.Body).Decode(&kuzObject); err != nil {
		msg := "error decoding expected SON object from the HTTP request body"
		serverLog.Error(err, msg)
		return nil, nil, errors.New(msg)
	} else if crBytes, err := base64.StdEncoding.DecodeString(kuzObject.Cr); err != nil {
		msg := "error base64 decoding cr"
		serverLog.Error(err, msg)
		return nil, nil, errors.New(msg)
	} else if configTarZipBytes, err := base64.StdEncoding.DecodeString(kuzObject.Config); err != nil {
		msg := "error base64 decoding config"
		serverLog.Error(err, msg)
		return nil, nil, errors.New(msg)
	} else {
		return crBytes, configTarZipBytes, nil
	}
}

func stageKuzRequest(crBytes []byte, configTarZipBytes []byte) (configDir string, cleanup func(), err error) {
	tmpDir, err := ioutil.TempDir("", "test_kuz_server")
	if err != nil {
		serverLog.Error(err, "error creating tmp directory")
		return "", nil, err
	}
	defer func() {
		if err != nil {
			_ = os.RemoveAll(tmpDir)
		}
	}()

	configArchive := filepath.Join(tmpDir, "config.tgz")
	configDir = filepath.Join(tmpDir, "config")

	if err = ioutil.WriteFile(filepath.Join(tmpDir, "CR.yaml"), crBytes, os.ModePerm); err != nil {
		serverLog.Error(err, "error writing CR.yaml to tmp directory")
		return "", nil, err
	} else if err = ioutil.WriteFile(configArchive, configTarZipBytes, os.ModePerm); err != nil {
		serverLog.Error(err, "error writing config.tgz to tmp directory")
		return "", nil, err
	} else if err = os.MkdirAll(configDir, os.ModePerm); err != nil {
		serverLog.Error(err, "error creating config directory")
		return "", nil, err
	} else if err = archiver.NewTarGz().Unarchive(configArchive, configDir); err != nil {
		serverLog.Error(err, fmt.Sprintf("error uncompressing: %v to %v", configArchive, configDir))
		return "", nil, err
	}

	return configDir, func() {
		_ = os.RemoveAll(tmpDir)
	}, nil
}

func createKuzK8sService(ctx context.Context, cfg *rest.Config, port int32) (*v1.Service, error) {
	servicePorts := []v1.ServicePort{
		{
			Port:     port,
			Name:     kuzPortName,
			Protocol: v1.ProtocolTCP,
			TargetPort: intstr.IntOrString{
				Type:   intstr.Int,
				IntVal: port,
			},
		},
	}
	client, err := crclient.New(cfg, crclient.Options{})
	if err != nil {
		return nil, fmt.Errorf("failed to create new client: %w", err)
	}
	s, err := initOperatorService(ctx, client, servicePorts)
	if err != nil {
		if err == k8sutil.ErrNoNamespace || err == k8sutil.ErrRunLocal {
			serverLog.Info("Skipping kuztomize build Service creation; not running in a cluster.")
			return nil, nil
		}
		return nil, fmt.Errorf("failed to initialize service object for kustomize: %w", err)
	}
	service, err := createOrUpdateService(ctx, client, s)
	if err != nil {
		return nil, fmt.Errorf("failed to create or get service for kustomize: %w", err)
	}

	return service, nil
}

func createOrUpdateService(ctx context.Context, client crclient.Client, s *v1.Service) (*v1.Service, error) {
	if err := client.Create(ctx, s); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return nil, err
		}
		// Service already exists, we want to update it
		// as we do not know if any fields might have changed.
		existingService := &v1.Service{}
		err := client.Get(ctx, types.NamespacedName{
			Name:      s.Name,
			Namespace: s.Namespace,
		}, existingService)
		if err != nil {
			return nil, err
		}

		s.ResourceVersion = existingService.ResourceVersion
		if existingService.Spec.Type == v1.ServiceTypeClusterIP {
			s.Spec.ClusterIP = existingService.Spec.ClusterIP
		}
		err = client.Update(ctx, s)
		if err != nil {
			return nil, err
		}
		serverLog.Info("kustomize Service object updated", "Service.Name",
			s.Name, "Service.Namespace", s.Namespace)
		return s, nil
	}

	serverLog.Info("Kustomize Service object created", "Service.Name",
		s.Name, "Service.Namespace", s.Namespace)
	return s, nil
}

func initOperatorService(ctx context.Context, client crclient.Client, sp []v1.ServicePort) (*v1.Service, error) {
	operatorName, err := k8sutil.GetOperatorName()
	if err != nil {
		return nil, err
	}
	namespace, err := k8sutil.GetOperatorNamespace()
	if err != nil {
		return nil, err
	}
	label := map[string]string{"name": operatorName}

	service := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-kuztomize", operatorName),
			Namespace: namespace,
			Labels:    label,
		},
		Spec: v1.ServiceSpec{
			Ports:    sp,
			Selector: label,
		},
	}

	ownRef, err := getPodOwnerRef(ctx, client, namespace)
	if err != nil {
		return nil, err
	}
	service.SetOwnerReferences([]metav1.OwnerReference{*ownRef})

	return service, nil
}

func getPodOwnerRef(ctx context.Context, client crclient.Client, ns string) (*metav1.OwnerReference, error) {
	// Get current Pod the operator is running in
	pod, err := k8sutil.GetPod(ctx, client, ns)
	if err != nil {
		return nil, err
	}
	podOwnerRefs := metav1.NewControllerRef(pod, pod.GroupVersionKind())
	// Get Owner that the Pod belongs to
	ownerRef := metav1.GetControllerOf(pod)
	finalOwnerRef, err := findFinalOwnerRef(ctx, client, ns, ownerRef)
	if err != nil {
		return nil, err
	}
	if finalOwnerRef != nil {
		return finalOwnerRef, nil
	}

	// Default to returning Pod as the Owner
	return podOwnerRefs, nil
}

// findFinalOwnerRef tries to locate the final controller/owner based on the owner reference provided.
func findFinalOwnerRef(ctx context.Context, client crclient.Client, ns string,
	ownerRef *metav1.OwnerReference) (*metav1.OwnerReference, error) {
	if ownerRef == nil {
		return nil, nil
	}

	obj := &unstructured.Unstructured{}
	obj.SetAPIVersion(ownerRef.APIVersion)
	obj.SetKind(ownerRef.Kind)
	err := client.Get(ctx, types.NamespacedName{Namespace: ns, Name: ownerRef.Name}, obj)
	if err != nil {
		return nil, err
	}
	newOwnerRef := metav1.GetControllerOf(obj)
	if newOwnerRef != nil {
		return findFinalOwnerRef(ctx, client, ns, newOwnerRef)
	}

	serverLog.V(1).Info("Pods owner found", "Kind", ownerRef.Kind, "Name",
		ownerRef.Name, "Namespace", ns)
	return ownerRef, nil
}
