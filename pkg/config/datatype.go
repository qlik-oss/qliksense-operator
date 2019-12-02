package config

import (
	"sigs.k8s.io/kustomize/api/types"
)

// CRConfig defines the configuration for the whole manifests
// It is expecting in the manifestsRoot folder two subfolders .operator and .configuration exist
// operator will add patch into .operator folder
// customer will add patch into .configuration folder
type CRConfig struct {
	// relative to manifestsRoot folder, ex. ./manifests/base
	ConfigProfile    string   `json:"configProfile" yaml:"configProfile"`
	Secrets          []Secret `json:"secrets,omitempty" yaml:"secrets,omitempty"`
	Configs          []Config `json:"configs,omitempty" yaml:"configs,omitempty"`
	ManifestsRoot    string   `json:"manifestsRoot,omitempty" yaml:"manifestsRoot,omitempty"`
	GenerateKeys     bool     `json:"generateKeys,omitempty" yaml:"generateKeys,omitempty"`
	StorageClassName string   `json:"storageClassName,omitempty" yaml:"storageClassName,omitempty"`
}
type Secret struct {
	SecretKey string            `json:"secretKey" yaml:"secretKey"`
	Values    map[string]string `json:"values" yaml:"values"`
}

type Config struct {
	DataKey string            `json:"dataKey" yaml:"dataKey"`
	Values  map[string]string `json:"values" yaml:"values"`
}

type SelectivePatch struct {
	ApiVersion string            `yaml:"apiVersion"`
	Kind       string            `yaml:"kind"`
	Metadata   map[string]string `yaml:"metadata"`
	Enabled    bool              `yaml:"enabled,omitempty"`
	Patches    []types.Patch     `yaml:"patches,omitempty"`
}

type SupperConfigMap struct {
	ApiVersion string            `yaml:"apiVersion"`
	Kind       string            `yaml:"kind"`
	Metadata   map[string]string `yaml:"metadata,omitempty"`
	Data       map[string]string `yaml:"data,omitempty"`
}
type SupperSecret struct {
	ApiVersion string            `yaml:"apiVersion"`
	Kind       string            `yaml:"kind"`
	Metadata   map[string]string `yaml:"metadata,omitempty"`
	Data       map[string]string `yaml:"data,omitempty"`
	StringData map[string]string `yaml:"stringData,omitempty"`
}
