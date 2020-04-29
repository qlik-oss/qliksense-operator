package main

import (
	"context"
	"fmt"

	"net/http"

	"github.com/gorilla/mux"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/rest"
	crclient "sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	kuzServerHost       = "0.0.0.0"
	KuzPort       int32 = 7000
	kuzPortName         = "kuz-port"
)
var serverLog = logf.Log.WithName("kuz_server")

func addKuzServer(ctx context.Context, cfg *rest.Config, namespace string) {
	go startHTTPServer()
	servicePorts := []v1.ServicePort{
		{Port: KuzPort, Name: kuzPortName, Protocol: v1.ProtocolTCP, TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: KuzPort}},
	}

	_, err := createKuzService(ctx, cfg, servicePorts)
	if err != nil {
		serverLog.Info("Could not create kustomize Service", "error", err.Error())
	}

}

func Ping(rw http.ResponseWriter, r *http.Request) {
	fmt.Fprint(rw, "Pong")
}

// StartHTTPServer starts the server in port 8000 using mux router
func startHTTPServer() {
	r := mux.NewRouter()
	r.HandleFunc("/ping", Ping)
	address := fmt.Sprintf("%s:%d", kuzServerHost, KuzPort)
	serverLog.Info("Starting ku server server on 7000")
	http.ListenAndServe(address, r)
}

func createKuzService(ctx context.Context, cfg *rest.Config, servicePorts []v1.ServicePort) (*v1.Service, error) {
	if len(servicePorts) < 1 {
		return nil, fmt.Errorf("failed to create kuztomize build Serice; service ports were empty")
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
