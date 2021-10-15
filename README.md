# Docker Credentials from the Environment

A docker credential helper to streamline repository interactions in CI/CD pipelines, particularly Jenkins declarative Pipelines, where dynamic credentials are used.

## Environment Variables

For the docker repository `https://repo.example.com/v1`, the credential helper expects to retrieve credentials from the following environment variables:

* `DOCKER_repo_example_com_USR` containing the repository username
* `DOCKER_repo_example_com_PSW` containing the repository password, token or secret.

If no environment variables for the target repository's FQDN is found, then:

1. The helper will remove DNS labels from the FQDN one-at-a-time from the right, and look again, for example:
`DOCKER_repo_example_com_USR` => `DOCKER_example_com_USR` => `DOCKER_com_USR` => `DOCKER__USR`.
2. If the target repository is a private AWS ECR repository (FQDN matches the regex `^[0-9]+\.dkr\.ecr\.[-a-z0-9]+\.amazonaws\.com$`), it will attempt to exchange local AWS credentials (most likely exposed through `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY` environment variables) for short-lived ECR login credentials.

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
environment {
    DOCKER_hub_docker_com          = credentials('hub.docker.com') // Username-Password credential
    DOCKER_artifactory_example_com = credentials('jenkins.artifactory') // (Vault) Username-Password credential
    AWS_ROLE_ARN                   = 'arn:aws:iam::123456789:role/jenkins-user'
    // or perhaps AWS_CONFIG and AWS_PROFILE
    AWS_ACCESS_KEY_ID              = credentials('AWS_ACCESS_KEY_ID') // String credential
    AWS_SECRET_ACCESS_KEY          = credentials('AWS_SECRET_ACCESS_KEY') // String credential
}

stages {
    stage("Upload Image1 to Docker Hub") {
        steps {
            sh "docker push hub.docker.com/example/example-image:1.0"
        }
    }

    stage("Upload Image2 to Artifactory") {
        steps {
            sh "docker push artifactory.example.com/example/example-image:1.0"
        }
    }

    stage("Upload Image3 to AWS-ECR") {
        steps {
            sh "docker push 123456789.dkr.ecr.us-east-1.amazonaws.com/example/example-image:1.0"
        }
    }
}
```
