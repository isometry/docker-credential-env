package provider

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
)

// AccountRegionEnv retrieves AWS credentials from environment variables
// that are prefixed with a specific AWS account ID and region.
//
// For example, if AccountID is "123456789012" and Region is "us-east-1",
// it will look for environment variables like:
// - AWS_ACCESS_KEY_ID_123456789012_us_east_1
// - AWS_SECRET_ACCESS_KEY_123456789012_us_east_1
// - AWS_SESSION_TOKEN_123456789012_us_east_1 (optional)
//
// Note: Hyphens in the region name are replaced with underscores for the environment variable name.
type AccountRegionEnv struct {
	Hostname  string
	AccountID string
	Region    string
}

// Retrieve fetches the credentials.
// This method is part of the aws.CredentialsProvider interface.
func (p *AccountRegionEnv) Retrieve(_ context.Context) (out aws.Credentials, err error) {
	if p.AccountID == "" || p.Region == "" {
		return aws.Credentials{}, fmt.Errorf("AccountRegionEnv: AccountID and Region must be set")
	}

	defer func() {
		// Diagnostic output
		if out.Source != "" {
			_, _ = fmt.Fprintf(os.Stderr, "Authenticating access to %q with %q", p.Hostname, out.Source)
		}
	}()

	// Construct the suffix for the environment variables.
	// Replace hyphens in region with underscores as environment variables typically don't use hyphens.
	envRegion := strings.ReplaceAll(p.Region, "-", "_")
	suffix := fmt.Sprintf("_%s_%s", p.AccountID, strings.ToLower(envRegion))

	// Check for suffixed environment variables
	accessKeyID := os.Getenv("AWS_ACCESS_KEY_ID" + suffix)
	secretAccessKey := os.Getenv("AWS_SECRET_ACCESS_KEY" + suffix)
	sessionToken := os.Getenv("AWS_SESSION_TOKEN" + suffix)

	// If ANY suffixed credentials exist, require ALL mandatory suffixed credentials
	if accessKeyID != "" || secretAccessKey != "" || sessionToken != "" {
		// If using suffixed credentials, both access key and secret key must be present
		if accessKeyID == "" {
			return aws.Credentials{}, fmt.Errorf("AccountRegionEnv: environment variable %s not found", "AWS_ACCESS_KEY_ID"+suffix)
		}
		if secretAccessKey == "" {
			return aws.Credentials{}, fmt.Errorf("AccountRegionEnv: environment variable %s not found", "AWS_SECRET_ACCESS_KEY"+suffix)
		}

		// Use only the suffixed credentials
		out = aws.Credentials{
			AccessKeyID:     accessKeyID,
			SecretAccessKey: secretAccessKey,
			SessionToken:    sessionToken, // Session token is optional, can be empty
			Source:          fmt.Sprintf("AccountRegionEnv (Account: %s, Region: %s)", p.AccountID, p.Region),
		}
		return out, nil
	}

	// No suffixed credentials found, fall back to standard AWS credentials
	accessKeyID = os.Getenv("AWS_ACCESS_KEY_ID")
	secretAccessKey = os.Getenv("AWS_SECRET_ACCESS_KEY")
	sessionToken = os.Getenv("AWS_SESSION_TOKEN")

	// Check if standard credentials are available
	if accessKeyID == "" {
		return aws.Credentials{}, fmt.Errorf("AccountRegionEnv: no account/region credentials found and standard AWS_ACCESS_KEY_ID not found")
	}
	if secretAccessKey == "" {
		return aws.Credentials{}, fmt.Errorf("AccountRegionEnv: no account/region credentials found and standard AWS_SECRET_ACCESS_KEY not found")
	}

	out = aws.Credentials{
		AccessKeyID:     accessKeyID,
		SecretAccessKey: secretAccessKey,
		SessionToken:    sessionToken, // Session token is optional, can be empty
		Source:          fmt.Sprintf("Standard AWS Environment (Account: %s, Region: %s)", p.AccountID, p.Region),
	}
	return out, nil
}
