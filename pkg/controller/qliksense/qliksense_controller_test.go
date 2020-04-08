package qliksense

import (
	"context"

	"errors"
	"github.com/qlik-oss/qliksense-operator/tests"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"testing"

	qlikv1 "github.com/qlik-oss/qliksense-operator/pkg/apis/qlik/v1"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

func TestAdd(t *testing.T) {
	tests := []struct {
		name                string
		setupMockManager    func() (*tests.MockKubernetesControllerManager, *tests.MockKubernetesFieldIndexer)
		reconciler          reconcile.Reconciler
		setupMockController func() *tests.MockKubernetesController

		expectedErr error
	}{
		{
			name: "returns err when ctrlbuilder returns error",
			setupMockManager: func() (*tests.MockKubernetesControllerManager, *tests.MockKubernetesFieldIndexer) {
				mi := &tests.MockKubernetesFieldIndexer{}
				m := &tests.MockKubernetesControllerManager{}
				scheme := generateScheme(t, &qlikv1.Qliksense{}, &qlikv1.QliksenseList{})
				client := &tests.MockKubernetesControllerClient{}
				m.On("GetClient").Return(scheme)
				m.On("GetScheme").Return(client)
				return m, mi
			},
			reconciler: &tests.MockKubernetesControllerReconciler{},
			setupMockController: func() *tests.MockKubernetesController {
				mockController := &tests.MockKubernetesController{}
				return mockController
			},
			expectedErr: errors.New("test controller builder error"),
		},
		{
			name: "returns err when ctrlbuilder returns error",
			setupMockManager: func() (*tests.MockKubernetesControllerManager, *tests.MockKubernetesFieldIndexer) {
				mi := &tests.MockKubernetesFieldIndexer{}
				m := &tests.MockKubernetesControllerManager{}
				m.On("GetFieldIndexer").Return(mi)
				mi.On("IndexField", &qlikv1.Qliksense{}, "spec", mock.Anything).Return(errors.New("test indexField error"))
				scheme := generateScheme(t, &qlikv1.Qliksense{}, &qlikv1.QliksenseList{})
				client := &tests.MockKubernetesControllerClient{}
				m.On("GetClient").Return(scheme)
				m.On("GetScheme").Return(client)
				return m, mi
			},
			reconciler: &tests.MockKubernetesControllerReconciler{},
			setupMockController: func() *tests.MockKubernetesController {
				mockController := &tests.MockKubernetesController{}
				return mockController
			},
			expectedErr: errors.New("test indexField error"),
		},
		{
			name: "returns err when controller.Watch returns error",
			setupMockManager: func() (*tests.MockKubernetesControllerManager, *tests.MockKubernetesFieldIndexer) {
				mi := &tests.MockKubernetesFieldIndexer{}
				m := &tests.MockKubernetesControllerManager{}
				m.On("GetFieldIndexer").Return(mi)
				mi.On("IndexField", &qlikv1.Qliksense{}, "spec", mock.Anything).Return(nil)
				scheme := generateScheme(t, &qlikv1.Qliksense{}, &qlikv1.QliksenseList{})
				client := &tests.MockKubernetesControllerClient{}
				m.On("GetClient").Return(scheme)
				m.On("GetScheme").Return(client)
				return m, mi
			},
			reconciler: &tests.MockKubernetesControllerReconciler{},
			setupMockController: func() *tests.MockKubernetesController {
				mockController := &tests.MockKubernetesController{}
				mockController.On("Watch", &source.Kind{Type: &qlikv1.Qliksense{}}, mock.Anything, mock.Anything).Return(errors.New("test watch error"))
				return mockController
			},
			expectedErr: errors.New("test watch error"),
		},
		{
			name: "returns err when controller.Watch returns error",
			setupMockManager: func() (*tests.MockKubernetesControllerManager, *tests.MockKubernetesFieldIndexer) {
				mi := &tests.MockKubernetesFieldIndexer{}
				m := &tests.MockKubernetesControllerManager{}
				m.On("GetFieldIndexer").Return(mi)
				mi.On("IndexField", &qlikv1.Qliksense{}, "spec", mock.Anything).Return(nil)
				scheme := generateScheme(t, &qlikv1.Qliksense{}, &qlikv1.QliksenseList{})
				client := &tests.MockKubernetesControllerClient{}
				m.On("GetClient").Return(scheme)
				m.On("GetScheme").Return(client)
				return m, mi
			},
			reconciler: &tests.MockKubernetesControllerReconciler{},
			setupMockController: func() *tests.MockKubernetesController {
				mockController := &tests.MockKubernetesController{}
				mockController.On("Watch", &source.Kind{Type: &qlikv1.Qliksense{}}, mock.Anything, mock.Anything).Return(nil)
				return mockController
			},
			expectedErr: errors.New("test watch error"),
		},
		{
			name: "returns nil when controller.Watch returns no error",
			setupMockManager: func() (*tests.MockKubernetesControllerManager, *tests.MockKubernetesFieldIndexer) {
				mi := &tests.MockKubernetesFieldIndexer{}
				m := &tests.MockKubernetesControllerManager{}
				m.On("GetFieldIndexer").Return(mi)
				mi.On("IndexField", &qlikv1.Qliksense{}, "spec", mock.Anything).Return(nil)
				scheme := generateScheme(t, &qlikv1.Qliksense{}, &qlikv1.QliksenseList{})
				client := &tests.MockKubernetesControllerClient{}
				m.On("GetClient").Return(scheme)
				m.On("GetScheme").Return(client)
				return m, mi
			},
			reconciler: &tests.MockKubernetesControllerReconciler{},
			setupMockController: func() *tests.MockKubernetesController {
				mockController := &tests.MockKubernetesController{}
				mockController.On("Watch", &source.Kind{Type: &qlikv1.Qliksense{}}, mock.Anything, mock.Anything).Return(nil)
				return mockController
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockMgr, mockIndexer := tt.setupMockManager()
			mockController := tt.setupMockController()

			err := Add(mockMgr)
			if tt.expectedErr != nil {
				require.Error(t, err)
				require.Equal(t, tt.expectedErr, err)
			} else {
				require.NoError(t, err)
			}

			mockController.AssertExpectations(t)
			mockMgr.AssertExpectations(t)
			mockIndexer.AssertExpectations(t)
		})
	}
}

func generateScheme(t *testing.T, object ...runtime.Object) *runtime.Scheme {
	SchemeBuilder := &scheme.Builder{GroupVersion: schema.GroupVersion{Group: "qliksense.qlik.com", Version: "v1"}}
	bld := SchemeBuilder.Register(object...)
	scheme, err := bld.Build()
	require.NoError(t, err)
	return scheme
}



func TestAddFinalizer(t *testing.T) {

}

func TestSetupCronJob(t *testing.T) {

}

func TestReconcile(t *testing.T) {
	tests := []struct {
		name            string
		setupMockClient func() (*tests.MockKubernetesControllerClient, *tests.MockKubernetesControllerStatusWriter)
		mockRecorder    *tests.MockEventRecorder
		applyQliksenseCRs  func(ctx context.Context, qliksense *qlikv1.Qliksense, scheme *runtime.Scheme) error
		request         reconcile.Request
		expectedResult  reconcile.Result
		expectedErr     error
	}{
		{
			name: "qliksense resource deleted",
			setupMockClient: func() (*tests.MockKubernetesControllerClient, *tests.MockKubernetesControllerStatusWriter) {
				msw := &tests.MockKubernetesControllerStatusWriter{}
				m := &tests.MockKubernetesControllerClient{}
				msw.On("Update", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()
				m.On("Get", mock.Anything, client.ObjectKey{Name: "deleted-qliksense", Namespace: "namespace"}, mock.Anything).Return(
					k8serrors.NewNotFound(schema.GroupResource{Group: "blah", Resource: "blah"}, "blah"),
				)
				m.On("Status").Return(msw)
				return m, msw
			},
			request:        reconcile.Request{NamespacedName: types.NamespacedName{Name: "deleted-qliksense", Namespace: "namespace"}},
			expectedResult: reconcile.Result{Requeue: false},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient, teststatusWriterClient := tt.setupMockClient()


			
			scheme := generateScheme(t, &qlikv1.Qliksense{}, &qlikv1.QliksenseList{})
			r := &ReconcileQliksense{
				client:   mockClient,
				scheme:   scheme,
			}
			result, err := r.Reconcile(tt.request)
			if tt.expectedErr != nil {
				require.Error(t, err)
				require.Equal(t, tt.expectedErr, err)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tt.expectedResult, result)

			mockClient.AssertExpectations(t)
			teststatusWriterClient.AssertExpectations(t)

		})
	}
}
