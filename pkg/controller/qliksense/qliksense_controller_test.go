package qliksense

import (
	"github.com/qlik-oss/qliksense-operator/tests"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"errors"
	"testing"

	qliksense "github.com/qlik-oss/qliksense-operator/pkg/apis/qlik/v1"
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
				scheme := generateScheme(t, &qliksense.Qliksense{}, &qliksense.QliksenseList{})
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
				mi.On("IndexField", &qliksense.Qliksense{}, "spec.engineVariantName", mock.Anything).Return(errors.New("test indexField error"))
				scheme := generateScheme(t, &qliksense.Qliksense{}, &qliksense.QliksenseList{})
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
				mi.On("IndexField", &qliksense.Qliksense{}, "spec.engineVariantName", mock.Anything).Return(nil)
				scheme := generateScheme(t, &qliksense.Qliksense{}, &qliksense.QliksenseList{})
				client := &tests.MockKubernetesControllerClient{}
				m.On("GetClient").Return(scheme)
				m.On("GetScheme").Return(client)
				return m, mi
			},
			reconciler: &tests.MockKubernetesControllerReconciler{},
			setupMockController: func() *tests.MockKubernetesController {
				mockController := &tests.MockKubernetesController{}
				mockController.On("Watch", &source.Kind{Type: &qliksense.Qliksense{}}, mock.Anything, mock.Anything).Return(errors.New("test watch error"))
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
				mi.On("IndexField", &qliksense.Qliksense{}, "spec.engineVariantName", mock.Anything).Return(nil)
				scheme := generateScheme(t, &qliksense.Qliksense{}, &qliksense.QliksenseList{})
				client := &tests.MockKubernetesControllerClient{}
				m.On("GetClient").Return(scheme)
				m.On("GetScheme").Return(client)
				return m, mi
			},
			reconciler: &tests.MockKubernetesControllerReconciler{},
			setupMockController: func() *tests.MockKubernetesController {
				mockController := &tests.MockKubernetesController{}
				mockController.On("Watch", &source.Kind{Type: &qliksense.Qliksense{}}, mock.Anything, mock.Anything).Return(nil)
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
				mi.On("IndexField", &qliksense.Qliksense{}, "spec.engineVariantName", mock.Anything).Return(nil)
				scheme := generateScheme(t, &qliksense.Qliksense{}, &qliksense.QliksenseList{})
				client := &tests.MockKubernetesControllerClient{}
				m.On("GetClient").Return(scheme)
				m.On("GetScheme").Return(client)
				return m, mi
			},
			reconciler: &tests.MockKubernetesControllerReconciler{},
			setupMockController: func() *tests.MockKubernetesController {
				mockController := &tests.MockKubernetesController{}
				mockController.On("Watch", &source.Kind{Type: &qliksense.Qliksense{}}, mock.Anything, mock.Anything).Return(nil)
				return mockController
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockMgr, mockIndexer := tt.setupMockManager()
			mockController := tt.setupMockController()

			err := add(mockMgr, tt.reconciler)
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
