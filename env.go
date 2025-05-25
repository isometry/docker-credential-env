package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/sts"

	credhelpers "github.com/docker/docker-credential-helpers/credentials"
)

var (
	ecrHostname  = regexp.MustCompile(`^(?P<account>[0-9]+)\.dkr\.ecr\.(?P<region>[-a-z0-9]+)\.amazonaws\.com$`)
	ghcrHostname = regexp.MustCompile(`^ghcr\.io$`)
)

const (
	defaultScheme     = "https://"
	envPrefix         = "DOCKER"
	envUsernameSuffix = "USR"
	envPasswordSuffix = "PSW"
	envSeparator      = "_"
	envIgnoreLogin    = "IGNORE_DOCKER_LOGIN"
	envDebugMode      = "DOCKER_CREDENTIAL_ENV_DEBUG"
)

const (
	envAwsAccessKeyID     = "AWS_ACCESS_KEY_ID"
	envAwsSecretAccessKey = "AWS_SECRET_ACCESS_KEY" // #nosec G101
	envAwsSessionToken    = "AWS_SESSION_TOKEN"     // #nosec G101
	envAwsRoleArn         = "AWS_ROLE_ARN"
)

// NotSupportedError represents an error indicating that the operation is not supported.
type NotSupportedError struct{}

func (m *NotSupportedError) Error() string {
	return "not supported"
}

// Env implements the Docker credentials Helper interface.
type Env struct{}

// Add implements the set verb.
func (*Env) Add(*credhelpers.Credentials) error {
	switch {
	case os.Getenv(envIgnoreLogin) != "":
		return nil
	default:
		return fmt.Errorf("add: %w", &NotSupportedError{})
	}
}

// Delete implements the erase verb.
func (*Env) Delete(string) error {
	switch {
	case os.Getenv(envIgnoreLogin) != "":
		return nil
	default:
		return fmt.Errorf("delete: %w", &NotSupportedError{})
	}
}

// List implements the list verb.
func (*Env) List() (map[string]string, error) {
	return nil, fmt.Errorf("list: %w", &NotSupportedError{})
}

// Get implements the get verb.
func (e *Env) Get(serverURL string) (username string, password string, err error) {
	var (
		hostname string
		ok       bool
	)

	hostname, err = getHostname(serverURL)
	if err != nil {
		return
	}

	if username, password, ok = getEnvCredentials(hostname); ok {
		return
	}

	submatches := ecrHostname.FindStringSubmatch(hostname)
	if submatches != nil {
		account := submatches[ecrHostname.SubexpIndex("account")]
		region := submatches[ecrHostname.SubexpIndex("region")]
		username, password, err = getEcrToken(account, region)
		return
	}

	if ghcrHostname.MatchString(hostname) {
		// This is a GitHub Container Registry: ghcr.io
		if token, found := os.LookupEnv("GITHUB_TOKEN"); found {
			username = "x-access-token"
			password = token
		}
		return
	}

	return
}

// getHostname extracts the hostname from the given server URL, adding a default scheme if missing, and returns it.
func getHostname(serverURL string) (hostname string, err error) {
	var server *url.URL
	server, err = url.Parse(defaultScheme + strings.TrimPrefix(serverURL, defaultScheme))
	if err != nil {
		return
	}

	hostname = server.Hostname()

	return
}

// getEnvVariables constructs environment variable names for username and password based on provided labels and offset.
// Returns the constructed environment variable names for the username and password.
func getEnvVariables(labels []string, offset int) (envUsername, envPassword string) {
	offset = max(0, min(offset, len(labels)))

	envHostname := strings.Join(labels[offset:], envSeparator)
	envUsername = strings.Join([]string{envPrefix, envHostname, envUsernameSuffix}, envSeparator)
	envPassword = strings.Join([]string{envPrefix, envHostname, envPasswordSuffix}, envSeparator)

	return
}

// getEnvCredentials retrieves credentials from environment variables based on the provided hostname.
// It parses the hostname, constructs environment variable names, and checks for corresponding values.
// Returns the username, password, and a boolean indicating if credentials were found.
func getEnvCredentials(hostname string) (username, password string, found bool) {
	hostname = strings.ReplaceAll(hostname, "-", "_")
	labels := strings.Split(hostname, ".")

	for i := 0; i <= len(labels); i++ {
		envUsername, envPassword := getEnvVariables(labels, i)

		if username, found = os.LookupEnv(envUsername); found {
			if password, found = os.LookupEnv(envPassword); found {
				break
			}
		}
	}
	return
}

// getEcrToken retrieves ECR authentication credentials (username and password) for the specified AWS account and hostname.
// It uses AWS SDK configuration with a custom retry mechanism (10 attempts max, 5 second max backoff)
// and a custom credentials provider that checks for account-specific environment variables.
// The ECR authorization token is retrieved with a 30 second timeout, decoded from base64,
// and split into username:password format. Debug mode will log token expiration time.
//
// Parameters:
//
//	hostname: The ECR repository hostname
//	account: The AWS account ID
//	region: The AWS region for the ECR repository
//
// Returns:
//
//	username: The decoded username (typically "AWS")
//	password: The decoded password token
//	err: Any error encountered during the process
func getEcrToken(account, region string) (username, password string, err error) {
	envProvider := &accountEnv{AccountID: account, Region: region}

	// Set up the AWS SDK config with a custom retryer
	simpleRetryer := func() aws.Retryer {
		standardRetryer := retry.NewStandard(func(options *retry.StandardOptions) {
			options.MaxAttempts = 10
			options.MaxBackoff = time.Second * 5
		})
		return retry.AddWithMaxBackoffDelay(standardRetryer, time.Second)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRetryer(simpleRetryer),
		config.WithRegion(region),
		config.WithCredentialsProvider(aws.NewCredentialsCache(envProvider)))
	if err != nil {
		return
	}

	var roleArn string
	if roleArn, err = getRoleArn(account, cfg.ConfigSources...); err != nil {
		return
	} else if roleArn != "" {
		stsSvc := sts.NewFromConfig(cfg)
		creds := stscreds.NewAssumeRoleProvider(stsSvc, roleArn)
		cfg.Credentials = aws.NewCredentialsCache(creds)
	}

	client := ecr.NewFromConfig(cfg)

	output, err := client.GetAuthorizationToken(ctx, nil)
	if err != nil {
		return
	}
	for _, authData := range output.AuthorizationData {
		if b, err := strconv.ParseBool(os.Getenv(envDebugMode)); err == nil && b {
			if authData.ExpiresAt != nil {
				expiration := authData.ExpiresAt.UTC().Format(time.RFC3339)
				_, _ = fmt.Fprintf(os.Stderr, "ECR token for %q will expire at %s (UTC)\n", account, expiration)
			}
		}

		if authData.AuthorizationToken == nil {
			err = fmt.Errorf("ecr: authorization token for %q is nil", account)
			return
		}

		var tokenBytes []byte
		tokenBytes, err = base64.StdEncoding.DecodeString(*authData.AuthorizationToken)
		if err != nil {
			return
		}
		token := bytes.SplitN(tokenBytes, []byte{':'}, 2)
		if len(token) != 2 {
			err = fmt.Errorf("ecr: invalid authorization token format for %q", account)
			return
		}

		username, password = string(token[0]), string(token[1])
	}
	return
}

// getRoleArn retrieves the AWS role ARN for a specific account by checking environment variables and AWS configurations.
// It checks the account-specific role ARN environment variable (AWS_ROLE_ARN_<account>). If not found,
// then checks the standard AWS role ARN environment variable (AWS_ROLE_ARN) when no config sources are provided.
// Finally, checks config sources which may contain role ARNs in AWS environment config or shared config.
// Returns role ARN string if found, empty string otherwise.
func getRoleArn(account string, configSources ...any) (roleARN string, err error) {
	val, found := os.LookupEnv(envAwsRoleArn + "_" + account)
	if found {
		return strings.TrimSpace(val), nil
	}

	// Check if any account-specific AWS credentials exist
	_, hasAccessKey := os.LookupEnv(envAwsAccessKeyID + "_" + account)
	_, hasSecretKey := os.LookupEnv(envAwsSecretAccessKey + "_" + account)
	if hasAccessKey || hasSecretKey {
		return "", fmt.Errorf("account-specific environment variables for %q are set, but no role ARN found", account)
	}

	if len(configSources) == 0 {
		return os.Getenv(envAwsRoleArn), nil
	}

	for _, x := range configSources {
		switch impl := x.(type) {
		case config.EnvConfig:
			if impl.RoleARN != "" {
				return strings.TrimSpace(impl.RoleARN), nil
			}
		case config.SharedConfig:
			if impl.RoleARN != "" {
				return strings.TrimSpace(impl.RoleARN), nil
			}
		}
	}
	return
}
