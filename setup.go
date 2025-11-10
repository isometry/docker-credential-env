package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/docker/cli/cli/config/configfile"
	"github.com/goccy/go-yaml"
)

// setupCmd handles the logic for the "setup" command.
type setupCmd struct {
	Command    string
	Out        io.Writer
	Registry   string
	configPath string
}

// Run executes the setup command.
func (c *setupCmd) Run() error {
	switch c.Command {
	case "show":
		return c.show()
	case "default":
		return c.configure(true)
	default:
		return c.configure(false)
	}
}

// show displays the current configuration.
func (c *setupCmd) show() error {
	config, err := c.loadConfig()
	if err != nil {
		return err
	}

	// Check if default credential store is set to 'env'
	defaultIsEnv := config.CredentialsStore == "env"

	// Collect registries that use 'env' credential helper
	var envRegistries []string
	if config.CredentialHelpers != nil {
		for registry, helper := range config.CredentialHelpers {
			if helper == "env" {
				envRegistries = append(envRegistries, registry)
			}
		}
	}
	slices.Sort(envRegistries)

	// Create output structure
	output := struct {
		Default    bool     `yaml:"default"`
		Registries []string `yaml:"registries"`
	}{
		Default:    defaultIsEnv,
		Registries: envRegistries,
	}

	// Marshal to YAML and output
	yamlData, err := yaml.MarshalWithOptions(&output, yaml.IndentSequence(true))
	if err != nil {
		return fmt.Errorf("failed to marshal output to YAML: %w", err)
	}
	_, err = fmt.Fprint(c.Out, string(yamlData))
	return err
}

// configure sets up the credential helper for a registry or as the default.
func (c *setupCmd) configure(defaultSetup bool) error {
	if !defaultSetup {
		if err := c.validateRegistry(); err != nil {
			return err
		}
	}

	if err := c.ensureDockerDir(); err != nil {
		return err
	}

	config, err := c.loadConfig()
	if err != nil {
		return err
	}

	// Check if already configured
	if (defaultSetup && config.CredentialsStore == "env") ||
		(!defaultSetup && config.CredentialHelpers[c.Registry] == "env") {
		if defaultSetup {
			_, err = fmt.Fprintln(c.Out, "Default credential store is already configured to use \"env\" credential helper")
		} else {
			_, err = fmt.Fprintf(c.Out, "Registry %q is already configured to use \"env\" credential helper\n", c.Registry)
		}
		return err
	}

	// Configure credential helper
	if defaultSetup {
		config.CredentialsStore = "env"
	} else {
		if config.CredentialHelpers == nil {
			config.CredentialHelpers = make(map[string]string)
		}
		config.CredentialHelpers[c.Registry] = "env"
	}

	// Save configuration
	if err = c.saveConfig(config); err != nil {
		return err
	}

	if defaultSetup {
		_, err = fmt.Fprintln(c.Out, "Default credential store successfully configured to use \"env\" credential helper")
	} else {
		_, err = fmt.Fprintf(c.Out, "Registry %q successfully configured to use \"env\" credential helper\n", c.Registry)
	}
	return err
}

func (c *setupCmd) validateRegistry() error {
	if c.Registry == "" {
		return errors.New("registry cannot be empty")
	}
	if strings.ContainsAny(c.Registry, " /\\") {
		return fmt.Errorf("invalid registry: %q", c.Registry)
	}
	return nil
}

func (c *setupCmd) ensureDockerDir() error {
	dockerDir := filepath.Dir(c.configPath)
	if err := os.MkdirAll(dockerDir, 0700); err != nil {
		return fmt.Errorf("failed to create Docker directory %q: %w", dockerDir, err)
	}
	return nil
}

func (c *setupCmd) loadConfig() (*configfile.ConfigFile, error) {
	configData, err := os.ReadFile(c.configPath)
	if os.IsNotExist(err) {
		return configfile.New(c.configPath), nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read Docker config file %q: %w", c.configPath, err)
	}
	var config configfile.ConfigFile
	if err := json.Unmarshal(configData, &config); err != nil {
		return nil, fmt.Errorf("failed to parse Docker config file %q: %w", c.configPath, err)
	}
	return &config, nil
}

func (c *setupCmd) saveConfig(config *configfile.ConfigFile) error {
	configData, err := json.MarshalIndent(config, "", "\t")
	if err != nil {
		return fmt.Errorf("failed to marshal Docker config: %w", err)
	}
	if err = os.WriteFile(c.configPath, configData, 0600); err != nil {
		return fmt.Errorf("failed to write Docker config file %q: %w", c.configPath, err)
	}
	return nil
}

// RunSetupCommand is the main entry point for the setup command.
func RunSetupCommand(args []string, out io.Writer) error {
	if len(args) < 1 {
		return errors.New("missing argument\nUsage: docker-credential-env setup <show|default|registry-url>")
	}

	cmd := &setupCmd{
		Command: args[0],
		Out:     out,
	}

	// Determine config path
	if dockerConfigDir := os.Getenv("DOCKER_CONFIG"); dockerConfigDir != "" {
		cmd.configPath = filepath.Join(dockerConfigDir, "config.json")
	} else {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get user home directory: %w", err)
		}
		cmd.configPath = filepath.Join(homeDir, ".docker", "config.json")
	}

	// Validate arguments
	switch cmd.Command {
	case "show", "default":
		if len(args) > 1 {
			return fmt.Errorf("%q command does not accept additional arguments", cmd.Command)
		}
	default: // Assumes registry
		cmd.Registry = args[0]
	}

	return cmd.Run()
}
