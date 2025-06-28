// docker-credential-env is a Docker credentials helper that reads credentials from
// the process environment.
//
// It supports both general environment variables (DOCKER_*_USR/PSW) and specialised
// credentials for AWS ECR and GitHub Container Registry.
//
// For more details, see the project README.md.

// Package main is the entry point for `docker-credential-env`.
package main

import (
	"fmt"
	"os"

	credhelpers "github.com/docker/docker-credential-helpers/credentials"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "setup" {
		// Extract arguments for setup command (skip program name and "setup")
		setupArgs := os.Args[2:]

		if err := RunSetupCommand(setupArgs, os.Stdout); err != nil {
			fmt.Fprintf(os.Stderr, "Setup failed: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// If not a setup command, serve as a credential helper
	credhelpers.Serve(&Env{})
}
