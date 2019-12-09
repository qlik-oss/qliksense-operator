package qust

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestUpdateSelectivePatchYaml(t *testing.T) {
	testCases := []struct {
		name                       string
		baseSelectivePatchYaml     string
		expectedSelectivePatchYaml string
	}{
		{
			name: "noPreviousData",
			baseSelectivePatchYaml: `apiVersion: qlik.com/v1
kind: SelectivePatch
metadata:
  name: keys-component-configs
enabled: true
patches:
- target:
    kind: SuperConfigMap
  patch: |-
    apiVersion: qlik.com/v1
    kind: SuperConfigMap
    metadata:
      name: keys-configs
`,
			expectedSelectivePatchYaml: `apiVersion: qlik.com/v1
kind: SelectivePatch
metadata:
  name: keys-component-configs
enabled: true
patches:
- target:
    kind: SuperConfigMap
  patch: |
    apiVersion: qlik.com/v1
    kind: SuperConfigMap
    metadata:
      name: keys-configs
    data:
      qlik.api.internal-foo: |
        (( index (ds "data") "foo" | base64.Decode ))
      qlik.api.internal-bar: |
        (( index (ds "data") "bar" | base64.Decode ))
`,
		},
		{
			name: "noPreviousData",
			baseSelectivePatchYaml: `apiVersion: qlik.com/v1
kind: SelectivePatch
metadata:
  name: keys-component-configs
enabled: true
patches:
- target:
    kind: SuperConfigMap
  patch: |-
    apiVersion: qlik.com/v1
    kind: SuperConfigMap
    metadata:
      name: keys-configs
    data:
      foo: bar
      baz: boo
`,
			expectedSelectivePatchYaml: `apiVersion: qlik.com/v1
kind: SelectivePatch
metadata:
  name: keys-component-configs
enabled: true
patches:
- target:
    kind: SuperConfigMap
  patch: |
    apiVersion: qlik.com/v1
    kind: SuperConfigMap
    metadata:
      name: keys-configs
    data:
      qlik.api.internal-foo: |
        (( index (ds "data") "foo" | base64.Decode ))
      qlik.api.internal-bar: |
        (( index (ds "data") "bar" | base64.Decode ))
`,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			transformedYamlBytes, err := updateSelectivePatchYaml([]byte(testCase.baseSelectivePatchYaml), []*serviceT{
				&serviceT{Name: "foo"},
				&serviceT{Name: "bar"},
			})
			assert.NoError(t, err)

			fmt.Printf("transformedYamlBytes: %v\n", string(transformedYamlBytes))
			assert.Equal(t, testCase.expectedSelectivePatchYaml, string(transformedYamlBytes))
		})
	}
}
