package main

import (
	"errors"
	"testing"
)

func TestGetHostname(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Full URL with scheme",
			input:    "https://example.com/path",
			expected: "example.com",
		},
		{
			name:     "Full URL without scheme",
			input:    "example.com/path",
			expected: "example.com",
		},
		{
			name:     "Simple domain with scheme",
			input:    "https://example.com",
			expected: "example.com",
		},
		{
			name:     "Simple domain without scheme",
			input:    "example.com",
			expected: "example.com",
		},
		{
			name:     "Full URL without scheme",
			input:    "example.com/path",
			expected: "example.com",
		},
		{
			name:     "Full URL with scheme and hyphen",
			input:    "https://example-hyphen.com/path",
			expected: "example-hyphen.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := getHostname(tt.input)
			if err != nil {
				t.Error(err)
			}
			if actual != tt.expected {
				t.Errorf("Get(%v) actual = (%v), expected (%v)", tt.input, actual, tt.expected)
			}
		})
	}
}

func TestGetEnvVariables(t *testing.T) {
	type args struct {
		labels []string
		offset int
	}

	type output struct {
		envUsername string
		envPassword string
	}

	tests := []struct {
		name     string
		input    args
		expected output
	}{
		{
			name:     "Negative Offset",
			input:    args{labels: []string{"repo", "example", "com"}, offset: -1},
			expected: output{envUsername: "DOCKER_repo_example_com_USR", envPassword: "DOCKER_repo_example_com_PSW"},
		},
		{
			name:     "Offset 0",
			input:    args{labels: []string{"repo", "example", "com"}, offset: 0},
			expected: output{envUsername: "DOCKER_repo_example_com_USR", envPassword: "DOCKER_repo_example_com_PSW"},
		},
		{
			name:     "Offset 1",
			input:    args{labels: []string{"repo", "example", "com"}, offset: 1},
			expected: output{envUsername: "DOCKER_example_com_USR", envPassword: "DOCKER_example_com_PSW"},
		},
		{
			name:     "Offset 2",
			input:    args{labels: []string{"repo", "example", "com"}, offset: 2},
			expected: output{envUsername: "DOCKER_com_USR", envPassword: "DOCKER_com_PSW"},
		},
		{
			name:     "Fallback",
			input:    args{labels: []string{"repo", "example", "com"}, offset: 3},
			expected: output{envUsername: "DOCKER__USR", envPassword: "DOCKER__PSW"},
		},
		{
			name:     "Overflow Offset",
			input:    args{labels: []string{"repo", "example", "com"}, offset: 4},
			expected: output{envUsername: "DOCKER__USR", envPassword: "DOCKER__PSW"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualEnvUsername, actualEnvPassword := getEnvVariables(tt.input.labels, tt.input.offset)
			if actualEnvUsername != tt.expected.envUsername || actualEnvPassword != tt.expected.envPassword {
				t.Errorf("Get(%v) actual = (%v, %v), expected (%v, %v)", tt.input, actualEnvUsername, actualEnvPassword, tt.expected.envUsername, tt.expected.envPassword)
			}
		})
	}

}

func TestGetEnvCredentials(t *testing.T) {
	type output struct {
		username string
		password string
		found    bool
	}

	tests := []struct {
		name     string
		input    string
		expected output
	}{
		{
			name:     "Exact match",
			input:    "example.com",
			expected: output{username: "u", password: "p", found: true},
		},
		{
			name:     "Subdomain",
			input:    "repo.example.com",
			expected: output{username: "u", password: "p", found: true},
		},
		{
			name:     "Hyphen-domain",
			input:    "repo-example.com",
			expected: output{username: "", password: "", found: false},
		},
		{
			name:     "Different domain",
			input:    "example.net",
			expected: output{username: "", password: "", found: false},
		},
	}

	t.Setenv("DOCKER_example_com_USR", "u")
	t.Setenv("DOCKER_example_com_PSW", "p")

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualUsername, actualPassword, actualFound := getEnvCredentials(tt.input)
			if actualUsername != tt.expected.username || actualPassword != tt.expected.password || actualFound != tt.expected.found {
				t.Errorf("getEnvCredentials(%v) actual = (%v, %v, %v), expected (%v, %v, %v)", tt.input, actualUsername, actualPassword, actualFound, tt.expected.username, tt.expected.password, tt.expected.found)
			}
		})
	}
}

func TestEnvGet(t *testing.T) {
	type output struct {
		username string
		password string
		err      error
	}

	tests := []struct {
		name     string
		input    string
		expected output
	}{
		{
			name:     "Domain with creds",
			input:    "https://example.com",
			expected: output{username: "u1", password: "p1", err: nil},
		},
		{
			name:     "Domain without creds",
			input:    "https://example.net",
			expected: output{username: "", password: "", err: nil},
		},
		{
			name:     "Subdomain with creds",
			input:    "https://repo.example.com",
			expected: output{username: "u2", password: "p2", err: nil},
		},
		{
			name:     "Subdomain without creds",
			input:    "https://other.example.com",
			expected: output{username: "u1", password: "p1", err: nil},
		},
		{
			name:     "Hyphen-domain with creds",
			input:    "https://repo-example.com",
			expected: output{username: "u2", password: "p2", err: nil},
		},
		{
			name:     "Hyphen-domain without creds",
			input:    "https://other-example.com",
			expected: output{username: "", password: "", err: nil},
		},
		{
			name:     "GitHub Container Registry",
			input:    "https://ghcr.io",
			expected: output{username: "x-access-token", password: "t1", err: nil},
		},
	}

	e := Env{}

	t.Setenv("DOCKER_example_com_USR", "u1")
	t.Setenv("DOCKER_example_com_PSW", "p1")
	t.Setenv("DOCKER_repo_example_com_USR", "u2")
	t.Setenv("DOCKER_repo_example_com_PSW", "p2")
	t.Setenv("GITHUB_TOKEN", "t1")

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualUsername, actualPassword, actualErr := e.Get(tt.input)
			if actualUsername != tt.expected.username || actualPassword != tt.expected.password || !errors.Is(actualErr, tt.expected.err) {
				t.Errorf("Get(%v) actual = (%v, %v, %v), expected (%v, %v, %v)", tt.input, actualUsername, actualPassword, actualErr, tt.expected.username, tt.expected.password, tt.expected.err)
			}
		})
	}
}

func TestEnvNotSupportedMethods(t *testing.T) {
	e := Env{}

	t.Run("Add is not supported", func(t *testing.T) {
		actualErr := e.Add(nil)
		if !errors.Is(actualErr, &NotSupportedError{}) {
			t.Errorf("Add() actual = (%v), expected (%v)", actualErr, &NotSupportedError{})
		}
	})

	t.Run("Add is ignored with IGNORE_DOCKER_LOGIN", func(t *testing.T) {
		t.Setenv(envIgnoreLogin, "1")
		actualErr := e.Add(nil)
		if actualErr != nil {
			t.Errorf("Add() actual = (%v), expected (%v)", actualErr, nil)
		}
	})

	t.Run("Delete is not supported", func(t *testing.T) {
		actualErr := e.Delete("")
		if !errors.Is(actualErr, &NotSupportedError{}) {
			t.Errorf("Delete() actual = (%v), expected (%v)", actualErr, &NotSupportedError{})
		}
	})

	t.Run("Delete is ignored with IGNORE_DOCKER_LOGIN", func(t *testing.T) {
		t.Setenv(envIgnoreLogin, "1")
		actualErr := e.Delete("")
		if actualErr != nil {
			t.Errorf("Delete() actual = (%v), expected (%v)", actualErr, nil)
		}
	})

	t.Run("List is not supported", func(t *testing.T) {
		_, actualErr := e.List()
		if !errors.Is(actualErr, &NotSupportedError{}) {
			t.Errorf("List() actual = (%v), expected (%v)", actualErr, &NotSupportedError{})
		}
	})
}

func TestGetRoleArn(t *testing.T) {
	tests := []struct {
		name     string
		inputEnv map[string]string
		expected string
	}{
		{
			name: "Standard environment variables",
			inputEnv: map[string]string{
				"AWS_ROLE_ARN": "arn:aws:iam::123456789012:role/my-role",
			},
			expected: "arn:aws:iam::123456789012:role/my-role",
		},
		{
			name: "Suffixed environment variables",
			inputEnv: map[string]string{
				"AWS_ROLE_ARN_123456789012": "arn:aws:iam::123456789012:role/my-role",
			},
			expected: "arn:aws:iam::123456789012:role/my-role",
		},
		{
			name: "Suffixed has higher priority",
			inputEnv: map[string]string{
				"AWS_ROLE_ARN":              "arn:aws:iam::123456789012:role/other-role",
				"AWS_ROLE_ARN_123456789012": "arn:aws:iam::123456789012:role/my-role",
			},
			expected: "arn:aws:iam::123456789012:role/my-role",
		},
		{
			name: "Suffixed variables set but role ARN set for standard environment",
			inputEnv: map[string]string{
				"AWS_ROLE_ARN":                       "arn:aws:iam::123456789012:role/my-role",
				"AWS_ACCESS_KEY_ID_123456789012":     "AKIA...",
				"AWS_SECRET_ACCESS_KEY_123456789012": "wJalr...",
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.inputEnv {
				t.Setenv(k, v)
			}
			actual := getRoleArn("123456789012")
			if actual != tt.expected {
				t.Errorf("GetRoleArn(<account_id>) actual = (%v), expected (%v)", actual, tt.expected)
			}
		})
	}
}

func TestGetProfile(t *testing.T) {
	tests := []struct {
		name     string
		inputEnv map[string]string
		expected string
	}{
		{
			name: "Standard environment variable",
			inputEnv: map[string]string{
				"AWS_PROFILE": "my-profile",
			},
			expected: "my-profile",
		},
		{
			name: "Suffixed environment variable",
			inputEnv: map[string]string{
				"AWS_PROFILE_12345": "my-profile",
			},
			expected: "my-profile",
		},
		{
			name: "Suffixed has higher priority",
			inputEnv: map[string]string{
				"AWS_PROFILE":       "other-profile",
				"AWS_PROFILE_12345": "my-profile",
			},
			expected: "my-profile",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.inputEnv {
				t.Setenv(k, v)
			}
			actual := getProfile("12345")
			if actual != tt.expected {
				t.Errorf("GetProfile(<suffix>) actual = (%v), expected (%v)", actual, tt.expected)
			}
		})
	}
}
