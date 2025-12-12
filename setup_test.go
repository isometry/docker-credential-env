package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/docker/cli/cli/config/configfile"
)

// setupTestEnvironment sets up a temporary directory for Docker config
// and ensures it's used for the duration of the test.
func setupTestEnvironment(t *testing.T) string {
	t.Helper()
	tempDir := t.TempDir()
	t.Setenv("DOCKER_CONFIG", tempDir)
	return tempDir
}

func TestRunSetupCommand_Errors(t *testing.T) {
	setupTestEnvironment(t)

	testCases := []struct {
		name        string
		args        []string
		errContains string
	}{
		{"no args", []string{}, "missing argument"},
		{"default with extra args", []string{"default", "extra"}, `"default" command does not accept additional arguments`},
		{"show with extra args", []string{"show", "extra"}, `"show" command does not accept additional arguments`},
		{"invalid registry", []string{"invalid/registry"}, "invalid registry"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			out := new(bytes.Buffer)
			err := RunSetupCommand(tc.args, out)
			if err == nil {
				t.Fatalf("Expected an error but got none")
			}
			if !strings.Contains(err.Error(), tc.errContains) {
				t.Errorf("Expected error to contain %q, but got %q", tc.errContains, err.Error())
			}
		})
	}
}

func TestRunSetupCommand_Show(t *testing.T) {
	tempDir := setupTestEnvironment(t)
	configPath := filepath.Join(tempDir, "config.json")
	out := new(bytes.Buffer)

	// Run command
	err := RunSetupCommand([]string{"show"}, out)
	if err != nil {
		t.Fatalf("Expected no error but got %v", err)
	}

	actual := out.String()
	expected := "default: false\nregistries: []\n"
	if actual != expected {
		t.Errorf("Expected output %q, but got %q", expected, actual)
	}

	// Test with a configured file
	config := &configfile.ConfigFile{
		CredentialsStore: "env",
		CredentialHelpers: map[string]string{
			"docker.io": "env",
		},
	}
	configData, err := json.MarshalIndent(config, "", "\t")
	if err != nil {
		t.Fatalf("Unexpected error marshaling config: %v", err)
	}
	err = os.WriteFile(configPath, configData, 0600)
	if err != nil {
		t.Fatalf("Unexpected error writing config file: %v", err)
	}

	out.Reset()

	err = RunSetupCommand([]string{"show"}, out)
	if err != nil {
		t.Fatalf("Expected no error but got %v", err)
	}

	actual = out.String()
	expected = "default: true\nregistries:\n  - docker.io\n"
	if actual != expected {
		t.Errorf("Expected output %q, but got %q", expected, actual)
	}
}

func TestRunSetupCommand_Default(t *testing.T) {
	tempDir := setupTestEnvironment(t)
	configPath := filepath.Join(tempDir, "config.json")
	out := new(bytes.Buffer)

	// Run setup default
	err := RunSetupCommand([]string{"default"}, out)
	if err != nil {
		t.Fatalf("RunSetupCommand() failed: %v", err)
	}

	// Verify config
	configData, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}
	var config configfile.ConfigFile
	err = json.Unmarshal(configData, &config)
	if err != nil {
		t.Fatalf("Failed to unmarshal config file: %v", err)
	}

	if config.CredentialsStore != "env" { //nolint:goconst // Redundant constant
		t.Errorf("Expected credsStore to be 'env', got %q", config.CredentialsStore)
	}
}

func TestRunSetupCommand_Registry(t *testing.T) {
	tempDir := setupTestEnvironment(t)
	configPath := filepath.Join(tempDir, "config.json")
	out := new(bytes.Buffer)

	// Run setup for a registry
	err := RunSetupCommand([]string{"docker.io"}, out)
	if err != nil {
		t.Fatalf("RunSetupCommand() failed: %v", err)
	}

	// Verify config
	configData, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}
	var config configfile.ConfigFile
	err = json.Unmarshal(configData, &config)
	if err != nil {
		t.Fatalf("Failed to unmarshal config file: %v", err)
	}

	if helper, ok := config.CredentialHelpers["docker.io"]; !ok || helper != "env" {
		t.Errorf("Expected credHelper for 'docker.io' to be 'env', got %q", helper)
	}
}
