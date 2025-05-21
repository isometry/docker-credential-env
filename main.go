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
	credhelpers "github.com/docker/docker-credential-helpers/credentials"
)

func main() {
	credhelpers.Serve(&Env{})
}
