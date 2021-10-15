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

The `docker-credential-env` binary must be installed to `$PATH`.
In `~/.docker/config.json`:

```json
{
  "credsStore": "env"
}
```

## Example Usage

```groovy

stages {
    stage("Upload Image to Docker Hub") {
        environment {
            DOCKER_hub_docker_com = credentials('hub.docker.com')
        }
        steps {
            sh "docker push ${image1}"
        }
    }
    stage("Upload Image to AWS-ECR") {
        steps {
            withCredentials([
                string(credentialsId: 'AWS_ACCESS_KEY_ID', variable: 'AWS_ACCESS_KEY_ID'),
                string(credentialsId: 'AWS_SECRET_ACCESS_KEY', variable: 'AWS_SECRET_ACCESS_KEY')
            ]) {
                sh "docker push ${image2}"
            }
        }
    }
}
```
