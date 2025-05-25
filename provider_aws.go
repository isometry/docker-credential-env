// Package main provides AWS credential provider implementations.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
)

// accountEnv retrieves AWS credentials from environment variables
// that are suffixed with a specific AWS account ID.
//
// For example, if AccountID is "123456789012",,
// it will look for environment variables like:
// - AWS_ACCESS_KEY_ID_123456789012
// - AWS_SECRET_ACCESS_KEY_123456789012
// - AWS_SESSION_TOKEN_123456789012 (optional).
type accountEnv struct {
	AccountID string
	Region    string
}

// Retrieve fetches AWS credentials either from account-specific environment variables
// or falls back to standard AWS environment variables if account-specific ones are not found.
// This method implements the aws.CredentialsProvider interface.
func (p *accountEnv) Retrieve(_ context.Context) (out aws.Credentials, err error) {
	if p.AccountID == "" {
		return aws.Credentials{}, errors.New("accountEnv: AccountID must be set")
	}

	defer func() {
		// Diagnostic output
		if out.Source != "" {
			if b, err := strconv.ParseBool(os.Getenv(envDebugMode)); err == nil && b {
				_, _ = fmt.Fprintf(os.Stderr, "Authenticating access to '%s.dkr.ecr.%s.amazonaws.com/' with %q\n", p.AccountID, p.Region, out.Source)
			}
		}
	}()

	// Construct the suffix for the environment variables.
	suffix := "_" + p.AccountID

	// Check for suffixed environment variables
	accessKeyID := os.Getenv(envAwsAccessKeyID + suffix)
	secretAccessKey := os.Getenv(envAwsSecretAccessKey + suffix)
	sessionToken := os.Getenv(envAwsSessionToken + suffix)

	// If ANY suffixed credentials exist, require ALL mandatory suffixed credentials
	if accessKeyID != "" || secretAccessKey != "" || sessionToken != "" {
		// If using suffixed credentials, both the access-key and secret key must be present
		if accessKeyID == "" {
			return aws.Credentials{}, fmt.Errorf("accountEnv: environment variable %s not found", envAwsAccessKeyID+suffix)
		}
		if secretAccessKey == "" {
			return aws.Credentials{}, fmt.Errorf("accountEnv: environment variable %s not found", envAwsSecretAccessKey+suffix)
		}

		// Use only the suffixed credentials
		out = aws.Credentials{
			AccessKeyID:     accessKeyID,
			SecretAccessKey: secretAccessKey,
			SessionToken:    sessionToken, // Session token is optional, can be empty
			Source:          fmt.Sprintf("Suffixed AWS Environment (Account: %s)", p.AccountID),
		}
		return out, nil
	}

	// No suffixed credentials found, fall back to standard AWS credentials
	accessKeyID = os.Getenv(envAwsAccessKeyID)
	secretAccessKey = os.Getenv(envAwsSecretAccessKey)
	sessionToken = os.Getenv(envAwsSessionToken)

	// Check if standard credentials are available
	if accessKeyID == "" {
		return aws.Credentials{}, errors.New("accountEnv: no account credentials found and standard AWS_ACCESS_KEY_ID not found")
	}
	if secretAccessKey == "" {
		return aws.Credentials{}, errors.New("accountEnv: no account credentials found and standard AWS_SECRET_ACCESS_KEY not found")
	}

	out = aws.Credentials{
		AccessKeyID:     accessKeyID,
		SecretAccessKey: secretAccessKey,
		SessionToken:    sessionToken,
		Source:          fmt.Sprintf("Standard AWS Environment (Account: %s)", p.AccountID),
	}
	return out, nil
}
