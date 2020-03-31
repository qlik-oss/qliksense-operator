package controller

import (
	"errors"
	"github.com/qlik-oss/qliksense-operator/pkg/tests"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"testing"

)

func TestAddToManager(t *testing.T) {
	tests := []struct {
		name     string
		addFuncs []func(manager.Manager) error
		manager  manager.Manager
		err      error
	}{
		{
			name: "should not return error when add funcs do not error",
			addFuncs: []func(manager.Manager) error{
				func(manager.Manager) error { return nil },
				func(manager.Manager) error { return nil },
			},
			manager: &tests.MockKubernetesControllerManager{},
			err:     nil,
		},
		{
			name: "should return an error when an add func errors",
			addFuncs: []func(manager.Manager) error{
				func(manager.Manager) error { return nil },
				func(manager.Manager) error { return errors.New("error") },
			},
			manager: &tests.MockKubernetesControllerManager{},
			err:     errors.New("error"),
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			AddToManagerFuncs = tc.addFuncs
			err := AddToManager(tc.manager)
			if err != nil {
				require.Error(t, err)
				require.Equal(t, tc.err, err)
			} else {
				require.Nil(t, err)
			}
		})
	}
}
