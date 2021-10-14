// docker-credentials-env is a Docker credentials helper that reads
// credentials from the process environment.
//
// Specifically, it expects to find environment variables with the prefix
// DOCKER_ followed by the requested hostname, and lastly the suffixes
// _USR for username and _PSW for password, such as
// DOCKER_hub_docker_com_USR and DOCKER_hub_docker_com_PSW for Docker Hub.
//
// In order to streamline interaction with (e.g.) AWS ECR, in addition to
// looking for the full registry hostname, it will also incrementally strip
// DNS label components from the right. For example, in search for credentials
// for the ECR registry at https://1234.dkr.ecr.us-east-1.amazonaws.com, the
// following environment variables will be searched in order:
//  - DOCKER_1234_dkr_ecr_us-east-1_amazonaws_com_USR
//  - DOCKER_dkr_ecr_us-east-1_amazonaws_com_USR
//  - DOCKER_ecr_us-east-1_amazonaws_com_USR
//  - DOCKER_us-east-1_amazonaws_com_USR
//  - DOCKER_amazonaws_com_USR
//  - DOCKER_com_USR
//  - DOCKER__USR
//
// This naming convention is intended to streamline use in Jenkins Pipelines:
// environment {
//   DOCKER_hub_docker_com = credentials('hub.docker.com')
// }

package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
)

var GitCommit = ""
var Version = "0.0.0"
var PreRelease = "dev"

var ecrHostname = regexp.MustCompile(`^[0-9]\.dkr\.ecr\.[-a-z0-9]+\.amazonaws\.com$`)

const (
	envPrefix         = "DOCKER"
	envUsernameSuffix = "USR"
	envPasswordSuffix = "PSW"
	envSeparator      = "_"
)

type Credential struct {
	Username string `json:"Username"`
	Password string `json:"Secret"`
}

func main() {
	args := os.Args[1:]
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: docker-credential-env get <hostname>")
		fmt.Fprintln(os.Stderr, "\nThis is a Docker credential helper, not intended to be run directly from a shell.")
		os.Exit(1)
	}

	switch args[0] {
	case "get":
		server, err := url.Parse(args[1])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to parse server address: %s\n", err)
			os.Exit(1)
		}
		credential, ok := lookupEnvCredential(server.Hostname())
		if !ok {
			// No credentials for a host is a non-error case; respond with an empty credentials object.
			os.Stdout.WriteString("{}\n")
			os.Exit(0)
		}
		resultJSON, err := json.Marshal(credential)
		if err != nil {
			// Should never happen
			fmt.Fprintf(os.Stderr, "Failed to serialize result: %s\n", err)
			os.Exit(1)
		}
		os.Stdout.Write(resultJSON)
		os.Stdout.WriteString("\n")
		os.Exit(0)

	default:
		fmt.Fprintf(os.Stderr, "The 'env' credential helper is not able to %s credentials.\n", args[0])
		os.Exit(1)
	}
}

// lookupEnvCredential searches the environment looking Docker registry credentials for hostname,
// stripping least-significant DNS labels on failure
func lookupEnvCredential(hostname string) (credential Credential, found bool) {
	labels := strings.Split(hostname, ".")
	for i := 0; i <= len(labels); i++ {
		envHostname := strings.Join(labels[i:], envSeparator)
		envUsername := strings.Join([]string{envPrefix, envHostname, envUsernameSuffix}, envSeparator)
		envPassword := strings.Join([]string{envPrefix, envHostname, envPasswordSuffix}, envSeparator)
		if credential.Username, found = os.LookupEnv(envUsername); found {
			if credential.Password, found = os.LookupEnv(envPassword); found {
				break
			}
		}
	}
	if !found && ecrHostname.MatchString(hostname) {
		// This is an AWS ECR Docker Registry: <account-id>.dkr.ecr.<region>.amazonaws.com
		region := labels[3]
		var err error
		if credential, err = getEcrToken(region); err == nil {
			found = true
		}
	}
	return
}

func getEcrToken(region string) (credential Credential, err error) {
	ctx := context.TODO()
	cfg, err := config.LoadDefaultConfig(ctx) // includes authentication-via-environment
	if err != nil {
		return
	}

	client := ecr.NewFromConfig(cfg)

	output, err := client.GetAuthorizationToken(ctx, nil)
	if err != nil {
		return
	}
	for _, authData := range output.AuthorizationData {
		// authData.AuthorizationToken is a base64-encoded username:password string,
		// where the username is always expected to be "AWS".
		tokenBytes, _ := base64.StdEncoding.DecodeString(*authData.AuthorizationToken)
		token := strings.SplitN(string(tokenBytes), ":", 1)
		credential.Username, credential.Password = token[0], token[1]
	}
	return
}
