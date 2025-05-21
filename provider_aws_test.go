package main

import (
	"errors"
	"testing"
)

func TestAccountEnv_Retrieve(t *testing.T) {
	useCases := []struct {
		name        string
		accountID   string
		envVars     map[string]string
		expectedErr error
	}{
		{
			name:      "Valid credentials",
			accountID: "123456789012",
			envVars: map[string]string{
				"AWS_ACCESS_KEY_ID_123456789012":     "AKIA...",
				"AWS_SECRET_ACCESS_KEY_123456789012": "wJalr...",
			},
		},
		{
			name:      "Valid credentials with session token",
			accountID: "123456789012",
			envVars: map[string]string{
				"AWS_ACCESS_KEY_ID_123456789012":     "AKIA...",
				"AWS_SECRET_ACCESS_KEY_123456789012": "wJalr...",
				"AWS_SESSION_TOKEN_123456789012":     "AQoEXAMPLEH4...",
			},
		},
		{
			name:        "Missing access key with session token present",
			accountID:   "123456789012",
			expectedErr: errors.New("accountEnv: environment variable AWS_ACCESS_KEY_ID_123456789012 not found"),
			envVars: map[string]string{
				"AWS_SESSION_TOKEN_123456789012":     "AQoEXAMPLEH4...",
				"AWS_SECRET_ACCESS_KEY_123456789012": "wJalr...",
			},
		},
		{
			name:        "Missing secret key with access key present",
			accountID:   "123456789012",
			expectedErr: errors.New("accountEnv: environment variable AWS_SECRET_ACCESS_KEY_123456789012 not found"),
			envVars: map[string]string{
				"AWS_ACCESS_KEY_ID_123456789012": "AKIA...",
			},
		},
		{
			name:        "Missing both keys - fallback to standard AWS credentials",
			accountID:   "123456789012",
			expectedErr: errors.New("accountEnv: no account credentials found and standard AWS_ACCESS_KEY_ID not found"),
		},
		{
			name:      "Valid credentials in FedRAMP",
			accountID: "123456789012",
			envVars: map[string]string{
				"AWS_ACCESS_KEY_ID_123456789012":     "AKIA...",
				"AWS_SECRET_ACCESS_KEY_123456789012": "wJalr...",
			},
		},
		{
			name:      "Standard AWS credentials when no suffixed vars exist",
			accountID: "123456789012",
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

			provider := &accountEnv{
				AccountID: tc.accountID,
			}

			creds, err := provider.Retrieve(t.Context())
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
				accessKeyVar := "AWS_ACCESS_KEY_ID_" + tc.accountID
				secretKeyVar := "AWS_SECRET_ACCESS_KEY_" + tc.accountID
				sessionTokenVar := "AWS_SESSION_TOKEN_" + tc.accountID

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
