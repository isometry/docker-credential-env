// docker-credentials-env is a Docker credentials helper that reads
// credentials from the process environment.

package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
)

var (
	version     = "0.0.0"
	commit      = "none"
	date        = "unknown"
	ecrHostname = regexp.MustCompile(`^[0-9]+\.dkr\.ecr\.[-a-z0-9]+\.amazonaws\.com$`)
)

const (
	envPrefix         = "DOCKER"
	envUsernameSuffix = "USR"
	envPasswordSuffix = "PSW"
	envSeparator      = "_"
)

type Credential struct {
	Username string `json:"Username"`
	Secret   string `json:"Secret"`
}

func main() {
	args := os.Args[1:]

	if len(args) == 0 {
		showUsage(0)
	}

	switch args[0] {
	case "--version":
		showVersion()

	case "get":
		payload, err := readPayload(os.Stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "No payload received: %s\n", err)
			os.Exit(1)
		}

		server, err := url.Parse(payload)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to parse server address payload: %s\n", err)
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

	case "store", "erase":
		fmt.Fprintf(os.Stderr, "%s is not implemented\n", args[0])
		os.Exit(0)

	default:
		fmt.Fprintf(os.Stderr, "The 'env' credential helper is not able to %s credentials.\n", args[0])
		os.Exit(1)
	}
}

func showVersion() {
	fmt.Printf("docker-credential-env %s (%s), %s\n", version, commit, date)
	os.Exit(0)
}

func showUsage(exitCode int) {
	fmt.Fprintln(os.Stderr, "This is a Docker credential helper, not intended to be run directly from a shell.")
	os.Exit(exitCode)
}

func readPayload(reader io.Reader) (payload string, err error) {
	var content []byte
	content, err = io.ReadAll(reader)
	if err != nil {
		return
	}
	payload = string(bytes.TrimSpace(content))
	return
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
			if credential.Secret, found = os.LookupEnv(envPassword); found {
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
	cfg.Region = region

	client := ecr.NewFromConfig(cfg)

	output, err := client.GetAuthorizationToken(ctx, nil)
	if err != nil {
		return
	}
	for _, authData := range output.AuthorizationData {
		// authData.AuthorizationToken is a base64-encoded username:password string,
		// where the username is always expected to be "AWS".
		var tokenBytes []byte
		tokenBytes, err = base64.StdEncoding.DecodeString(*authData.AuthorizationToken)
		if err != nil {
			return
		}
		token := strings.SplitN(string(tokenBytes), ":", 1)
		credential.Username, credential.Secret = token[0], token[1]
	}
	return
}
