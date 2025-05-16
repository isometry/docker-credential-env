package provider

import (
	"fmt"
	"strings"
	"testing"
)

func TestAccountRegionEnv_Retrieve(t *testing.T) {
	useCases := []struct {
		name        string
		accountID   string
		region      string
		envVars     map[string]string
		expectedErr error
	}{
		{
			name:      "Valid credentials",
			accountID: "123456789012",
			region:    "us-east-1",
			envVars: map[string]string{
				"AWS_ACCESS_KEY_ID_123456789012_us_east_1":     "AKIA...",
				"AWS_SECRET_ACCESS_KEY_123456789012_us_east_1": "wJalr...",
			},
		},
		{
			name:      "Valid credentials with session token",
			accountID: "123456789012",
			region:    "us-east-1",
			envVars: map[string]string{
				"AWS_ACCESS_KEY_ID_123456789012_us_east_1":     "AKIA...",
				"AWS_SECRET_ACCESS_KEY_123456789012_us_east_1": "wJalr...",
				"AWS_SESSION_TOKEN_123456789012_us_east_1":     "AQoEXAMPLEH4...",
			},
		},
		{
			name:        "Missing access key with session token present",
			accountID:   "123456789012",
			region:      "us-east-1",
			expectedErr: fmt.Errorf("AccountRegionEnv: environment variable AWS_ACCESS_KEY_ID_123456789012_us_east_1 not found"),
			envVars: map[string]string{
				"AWS_SESSION_TOKEN_123456789012_us_east_1":     "AQoEXAMPLEH4...",
				"AWS_SECRET_ACCESS_KEY_123456789012_us_east_1": "wJalr...",
			},
		},
		{
			name:        "Missing secret key with access key present",
			accountID:   "123456789012",
			region:      "us-east-1",
			expectedErr: fmt.Errorf("AccountRegionEnv: environment variable AWS_SECRET_ACCESS_KEY_123456789012_us_east_1 not found"),
			envVars: map[string]string{
				"AWS_ACCESS_KEY_ID_123456789012_us_east_1": "AKIA...",
			},
		},
		{
			name:        "Missing both keys - fallback to standard AWS credentials",
			accountID:   "123456789012",
			region:      "us-east-1",
			expectedErr: fmt.Errorf("AccountRegionEnv: no account/region credentials found and standard AWS_ACCESS_KEY_ID not found"),
		},
		{
			name:      "Valid credentials in FedRAMP",
			accountID: "123456789012",
			region:    "us-gov-west-1",
			envVars: map[string]string{
				"AWS_ACCESS_KEY_ID_123456789012_us_gov_west_1":     "AKIA...",
				"AWS_SECRET_ACCESS_KEY_123456789012_us_gov_west_1": "wJalr...",
			},
		},
		{
			name:      "Standard AWS credentials when no suffixed vars exist",
			accountID: "123456789012",
			region:    "us-east-1",
			envVars: map[string]string{
				"AWS_ACCESS_KEY_ID":     "STD-AKIA...",
				"AWS_SECRET_ACCESS_KEY": "STD-wJalr...",
			},
		},
	}
	for _, tc := range useCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set environment variables
			for k, v := range tc.envVars {
				t.Setenv(k, v)
			}

			provider := &AccountRegionEnv{
				AccountID: tc.accountID,
				Region:    tc.region,
			}

			creds, err := provider.Retrieve(nil)
			if tc.expectedErr != nil {
				if err == nil {
					t.Errorf("expected error but got none")
				} else if err.Error() != tc.expectedErr.Error() {
					t.Errorf("expected error %v but got %v", tc.expectedErr, err)
				}
				return
			}
			if err != nil && tc.expectedErr == nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if tc.expectedErr == nil {
				envRegion := strings.ReplaceAll(tc.region, "-", "_")
				accessKeyVar := fmt.Sprintf("AWS_ACCESS_KEY_ID_%s_%s", tc.accountID, envRegion)
				secretKeyVar := fmt.Sprintf("AWS_SECRET_ACCESS_KEY_%s_%s", tc.accountID, envRegion)
				sessionTokenVar := fmt.Sprintf("AWS_SESSION_TOKEN_%s_%s", tc.accountID, envRegion)

				// If we're testing standard AWS credentials fallback
				if _, hasAccessKey := tc.envVars[accessKeyVar]; !hasAccessKey {
					if creds.AccessKeyID != tc.envVars["AWS_ACCESS_KEY_ID"] {
						t.Errorf("expected standard access key %v but got %v", tc.envVars["AWS_ACCESS_KEY_ID"], creds.AccessKeyID)
					}
					if creds.SecretAccessKey != tc.envVars["AWS_SECRET_ACCESS_KEY"] {
						t.Errorf("expected standard secret key %v but got %v", tc.envVars["AWS_SECRET_ACCESS_KEY"], creds.SecretAccessKey)
					}
					return
				}

				// Normal suffixed credentials
				if creds.AccessKeyID != tc.envVars[accessKeyVar] {
					t.Errorf("expected access key %v but got %v", tc.envVars[accessKeyVar], creds.AccessKeyID)
				}
				if creds.SecretAccessKey != tc.envVars[secretKeyVar] {
					t.Errorf("expected secret key %v but got %v", tc.envVars[secretKeyVar], creds.SecretAccessKey)
				}
				if creds.SessionToken != "" && creds.SessionToken != tc.envVars[sessionTokenVar] {
					t.Errorf("expected session token %v but got %v", tc.envVars[sessionTokenVar], creds.SessionToken)
				}
			}
		})
	}
}
