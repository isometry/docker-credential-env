# Docker Credentials from the Environment

A docker credential helper to streamline repository interactions in CI/CD pipelines, particularly Jenkins declarative Pipelines, where dynamic credentials are used.

In addition to handling simple credentials, it also fully support private AWS ECR repositories, including full automatic cross-account sts:AssumeRole support.

## Environment Variables

For the docker repository `https://repo.example.com/v1`, the credential helper expects to retrieve credentials from the following environment variables:

* `DOCKER_repo_example_com_USR` containing the repository username
* `DOCKER_repo_example_com_PSW` containing the repository password, token or secret.

If no environment variables for the target repository's FQDN is found, then:

1. The helper will remove DNS labels from the FQDN one-at-a-time from the right, and look again, for example:
`DOCKER_repo_example_com_USR` => `DOCKER_example_com_USR` => `DOCKER_com_USR` => `DOCKER__USR`.
2. If the target repository is a private AWS ECR repository (FQDN matches the regex `^[0-9]+\.dkr\.ecr\.[-a-z0-9]+\.amazonaws\.com$`), it will attempt to exchange local AWS credentials (most likely exposed through `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY` environment variables) for short-lived ECR login credentials, including automatic sts:AssumeRole if `role_arn` is specified (e.g. via `AWS_ROLE_ARN`).

## Configuration

The `docker-credential-env` binary must be installed to `$PATH`, configured via `~/.docker/config.json`:

* Handle all docker authentication:

  ```json
  {
    "credsStore": "env"
  }
  ```

* Handle docker authentication for specific repositories:

  ```json
  {
    "credHelpers": {
      "artifactory.example.com": "env"
    }
  }
  ```

## Example Usage

### Jenkins

```groovy
stages {
    stage("Push Image to Artifactory") {
        environment {
            DOCKER_artifactory_example_com = credentials('jenkins.artifactory') // (Vault) Username-Password credential
        }
        steps {
            sh "docker push artifactory.example.com/example/example-image:1.0"
        }
    }

    stage("Push Image to Docker Hub") {
        environment {
            DOCKER_docker_com = credentials('hub.docker.com') // Username-Password credential, exploiting domain search
        }
        steps {
            sh "docker push hub.docker.com/example/example-image:1.0"
        }
    }

    stage("Push Image to AWS-ECR") {
        environment {
            // any standard AWS authentication mechanisms are supported
            AWS_ROLE_ARN          = 'arn:aws:iam::123456789:role/jenkins-user' // triggers automatic sts:AssumeRole
            // AWS_CONFIG_FILE    = file('AWS_CONFIG')
            // AWS_PROFILE        = 'jenkins'
            AWS_ACCESS_KEY_ID     = credentials('AWS_ACCESS_KEY_ID') // String credential
            AWS_SECRET_ACCESS_KEY = credentials('AWS_SECRET_ACCESS_KEY') // String credential
        }
        steps {
            sh "docker push 123456789.dkr.ecr.us-east-1.amazonaws.com/example/example-image:1.0"
        }
    }
}
```
