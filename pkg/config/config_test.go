package config

import (
	"io"
	"os"
	"strings"
	"testing"
)

func setup(t *testing.T) io.Reader {
	t.Parallel()
	sampleConfig := `
  configProfile: manifests/base
  manifestsRoot: "."
  configs:
  - dataKey: acceptEULA
    values:
      qliksense: "yes"`
	os.Setenv("YAML_CONF", sampleConfig)
	return strings.NewReader(sampleConfig)
}

func TestReadCRConfigFromFile(t *testing.T) {
	reader := setup(t)
	cfg, err := ReadCRConfigFromFile(reader)
	if err != nil {
		t.Fatalf("error reading config from file")
	}
	if cfg.Configs[0].DataKey != "acceptEULA" {
		t.Fail()
	}
	if cfg.Configs[0].Values["qliksense"] != "yes" {
		t.Fail()
	}
}

func TestReadCRConfigFromEnvYaml(t *testing.T) {
	os.Setenv("YAML_CONF", "")
	_, err := ReadCRConfigFromEnvYaml()
	if err == nil {
		t.Fail()
	}
	setup(t)
	cfg, err := ReadCRConfigFromEnvYaml()
	if err != nil {
		t.Fatalf("error reading config from env")
	}
	if cfg.Configs[0].DataKey != "acceptEULA" {
		t.Fail()
	}
	if cfg.Configs[0].Values["qliksense"] != "yes" {
		t.Fail()
	}
}
