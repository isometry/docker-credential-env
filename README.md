# Docker Credentials from the Environment

A [Docker credential helper](https://docs.docker.com/engine/reference/commandline/login/#credential-helpers) to streamline repository interactions in scenarios where the cacheing of credentials to `~/.docker/config.json` is undesirable, including CI/CD pipelines, or anywhere ephemeral credentials are used.

All OCI registry clients that support `~/.docker/config.json` are supported, including [`oras`](https://oras.land/), [`crane`](https://github.com/google/go-containerregistry/blob/main/cmd/crane/README.md), [`grype`](https://github.com/anchore/grype), etc.

In addition to handling basic username:password credentials, the credential helper also includes special support for:

* Amazon Elastic Container Registry (ECR) repositories using [standard AWS credentials](https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-envvars.html), including automatic cross-account role assumption.
* [GitHub Packages](https://ghcr.io/) via the common `GITHUB_TOKEN` environment variable.

## Environment Variables

For the docker repository `https://repo.example.com/v1`, the credential helper expects to retrieve credentials from the following environment variables:

* `DOCKER_repo_example_com_USR` containing the repository username
* `DOCKER_repo_example_com_PSW` containing the repository password, token or secret.

If no environment variables for the target repository's FQDN is found, then:

1. The helper will remove DNS labels from the FQDN one-at-a-time from the right, and look again, for example:
   `DOCKER_repo_example_com_USR` => `DOCKER_example_com_USR` => `DOCKER_com_USR` => `DOCKER__USR`.
2. If the target repository is a private AWS ECR repository (FQDN matches the regex `^[0-9]+\.dkr\.ecr\.[-a-z0-9]+\.amazonaws\.com$`):
* By default, it will attempt to exchange local AWS credentials (most likely exposed through `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY` environment variables) for short-lived ECR login credentials, including automatic sts:AssumeRole if `role_arn` is specified (e.g. via `AWS_ROLE_ARN`).
* **Account Suffixed Credentials**: The helper can also use AWS credentials from environment variables suffixed with a specific AWS Account ID. These credentials are expected to be in the format:
  * `AWS_ACCESS_KEY_ID_<account_id>`
  * `AWS_SECRET_ACCESS_KEY_<account_id>`
  * `AWS_SESSION_TOKEN_<account_id>` (optional)
  * `AWS_ROLE_ARN_<account_id>` (optional)
  * `AWS_PROFILE_<account_id>` (optional)

### AWS Profile Selection

The helper supports using AWS named profiles for authentication:

* `AWS_PROFILE`: Specifies which profile to use from your AWS shared configuration files. This is used when no account-specific credentials or profile is found.
* `AWS_PROFILE_<account_id>`: Account-specific profile selection. When accessing an ECR repository for a specific AWS account, you can set this environment variable to use a specific named profile from your AWS shared configuration files.

The profile selection follows this order of precedence:
1. Account-specific profile (`AWS_PROFILE_<account_id>`)
2. Standard AWS credentials for the specific account (if any account-specific credentials are found)
3. Standard AWS profile (`AWS_PROFILE`) if no account-specific settings are found

Important note: The helper will first look for account-suffixed AWS credentials (e.g. AWS_ACCESS_KEY_ID_123456789012).
If ANY account-suffixed credentials are found, even partially, the helper requires ALL mandatory credentials to be
present with that account suffix. Only if NO account-suffixed credentials exist will the helper fall back to using
standard AWS credentials (AWS_ACCESS_KEY_ID etc).

Hyphens within DNS labels are transformed to underscores (`s/-/_/g`) for credential lookup.

### Debug Mode

Set the environment variable `DOCKER_CREDENTIAL_ENV_DEBUG=true` to enable diagnostic output. When enabled, the helper will print information about credential sources to stderr, which can help troubleshoot authentication issues, especially with AWS ECR repositories.

## Configuration

The `docker-credential-env` binary must be installed to `$PATH`, and is enabled via `~/.docker/config.json` (or `$DOCKER_CONFIG/config.json` if the `DOCKER_CONFIG` environment variable is set):

The `docker-credential-env` binary includes a `setup` sub-command to help configure Docker to use the credential helper.

* Configure all Docker authentication to use the `env` credential helper:
  ```bash
  docker-credential-env setup default
  ```
  or
  ```json
  {
    "credsStore": "env"
  }
  ```

* Configure a specific registry to use the `env` credential helper:
  ```bash
  docker-credential-env setup artifactory.example.com
  docker-credential-env setup ghcr.io
  docker-credential-env setup 123456789012.dkr.ecr.us-east-1.amazonaws.com
  ```

  ```json
  {
    "credHelpers": {
      "artifactory.example.com": "env",
      "ghcr.io": "env",
      "123456789012.dkr.ecr.us-east-1.amazonaws.com": "env"
    }
  }
  ```

By default, attempts to explicitly `docker {login,logout}` for registries configured to use the `env` credential helper will generate an error. To ignore these errors, set the environment variable `IGNORE_DOCKER_LOGIN=1`.

* Show current configuration for the `env` credential helper:
  ```bash
  docker-credential-env setup show
  ```

The setup command respects the `DOCKER_CONFIG` environment variable for locating and updating the Docker client configuration file.

## Example Usage

### Jenkins

```groovy
stages {
    stage('Push Image to Artifactory') {
        environment {
            DOCKER_artifactory_example_com = credentials('jenkins.artifactory') // (Vault) Username-Password credential
        }
        steps {
            sh 'docker push artifactory.example.com/example/example-image:1.0'
        }
    }

    stage('Push Image to Docker Hub') {
        environment {
            DOCKER_docker_com = credentials('hub.docker.com') // Username-Password credential, exploiting domain search
        }
        steps {
            sh 'docker push hub.docker.com/example/example-image:1.0'
        }
    }

    stage('Push Image to AWS-ECR (Standard Credentials)') {
        environment {
            // any standard AWS authentication mechanisms are supported
            AWS_ROLE_ARN                = 'arn:aws:iam::123456789:role/jenkins-user' // triggers automatic sts:AssumeRole
            // AWS_CONFIG_FILE          = file('AWS_CONFIG')
            // AWS_PROFILE              = 'jenkins'
            AWS_ACCESS_KEY_ID           = credentials('AWS_ACCESS_KEY_ID') // String credential
            AWS_SECRET_ACCESS_KEY       = credentials('AWS_SECRET_ACCESS_KEY') // String credential
            DOCKER_CREDENTIAL_ENV_DEBUG = 'true' // Enable debug output for credential helper
        }
        steps {
            sh 'docker push 123456789.dkr.ecr.us-east-1.amazonaws.com/example/example-image:1.0'
        }
    }

    stage('Push Image to AWS-ECR (Account Suffixed Credentials)') {
        environment {
            // Make sure to include all required suffixed credentials
            AWS_ROLE_ARN_987654321098          = credentials('AWS_ROLE_ARN') // String credential
            AWS_ACCESS_KEY_ID_987654321098     = credentials('AWS_ACCESS_KEY_ID') // String credential
            AWS_SECRET_ACCESS_KEY_987654321098 = credentials('AWS_SECRET_ACCESS_KEY') // String credential
            // AWS_SESSION_TOKEN_987654321098  = credentials('AWS_SESSION_TOKEN') // Optional
            DOCKER_CREDENTIAL_ENV_DEBUG        = 'true' // Enable debug output for credential helper
        }
        steps {
            sh '''
              docker push 987654321098.dkr.ecr.eu-west-1.amazonaws.com/another-example/another-image:2.0
            '''
        }
    }

    stage('Push Image to AWS-ECR (Using Named Profiles)') {
      environment {
        // Using standard profile for one account
        AWS_PROFILE                    = 'default-profile'
        // Using account-specific profile for another account
        AWS_PROFILE_987654321098       = 'account-specific-profile'
        DOCKER_CREDENTIAL_ENV_DEBUG    = 'true' // Enable debug output for credential helper
      }
      steps {
        sh '''
            # Uses AWS_PROFILE_987654321098
            docker push 987654321098.dkr.ecr.eu-west-1.amazonaws.com/another-example/another-image:2.0

            # Uses AWS_PROFILE for a different account
            docker push 123456789.dkr.ecr.us-east-1.amazonaws.com/example/example-image:1.0
          '''
      }
    }

  stage('Push Image to GHCR') {
        environment {
            GITHUB_TOKEN = credentials('github') // String credential
        }
        steps {
            sh 'docker push ghcr.io/example/example-image:1.0'
        }
    }
}
```
