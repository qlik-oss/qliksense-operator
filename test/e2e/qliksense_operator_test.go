package e2e

import (
	goctx "context"
	"fmt"
	"testing"
	"time"

	"github.com/qlik-oss/qliksense-operator/pkg/apis"
	v1 "github.com/qlik-oss/qliksense-operator/pkg/apis/qlik/v1"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var (
	retryInterval        = time.Second * 5
	timeout              = time.Second * 60
	cleanupRetryInterval = time.Second * 1
	cleanupTimeout       = time.Second * 5
)

func TestQlikSenseOperator(t *testing.T) {
	qliksenseList := &v1.QliksenseList{}
	err := framework.AddToFrameworkScheme(apis.AddToScheme, qliksenseList)
	if err != nil {
		t.Fatalf("failed to add custom resource scheme to framework: %v", err)
	}
	// run subtests
	t.Run("qliksense-group", func(t *testing.T) {
		t.Run("Cluster", QliksenseCluster)
		t.Run("Cluster2", QliksenseCluster)
	})
}

func qliksenseDeployTest(t *testing.T, f *framework.Framework, ctx *framework.TestCtx) error {
	namespace, err := ctx.GetNamespace()
	if err != nil {
		return fmt.Errorf("could not get namespace: %v", err)
	}
	// create qliksense custom resource
	exampleQliksense := &v1.Qliksense{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "qliksense-test",
			Namespace: namespace,
		},
	}
	// use TestCtx's create helper to create the object and add a cleanup function for the new object
	err = f.Client.Create(goctx.TODO(), exampleQliksense, &framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	if err != nil {
		return err
	}

	err = f.Client.Get(goctx.TODO(), types.NamespacedName{Name: "qliksense-test", Namespace: namespace}, exampleQliksense)
	if err != nil {
		return err
	}
	return nil
}

func QliksenseCluster(t *testing.T) {
	t.Parallel()
	ctx := framework.NewTestCtx(t)
	defer ctx.Cleanup()
	err := ctx.InitializeClusterResources(&framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	if err != nil {
		t.Fatalf("failed to initialize cluster resources: %v", err)
	}
	t.Log("Initialized cluster resources")
	namespace, err := ctx.GetNamespace()
	if err != nil {
		t.Fatal(err)
	}
	// get global framework variables
	f := framework.Global
	// wait for qliksense-operator to be ready
	err = e2eutil.WaitForOperatorDeployment(t, f.KubeClient, namespace, "qliksense-operator", 1, retryInterval, timeout)
	if err != nil {
		t.Fatal(err)
	}

	if err = qliksenseDeployTest(t, f, ctx); err != nil {
		t.Fatal(err)
	}
}
