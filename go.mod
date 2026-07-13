module github.com/isometry/docker-credential-env

go 1.26.1

require (
	github.com/aws/aws-sdk-go-v2 v1.42.1
	github.com/aws/aws-sdk-go-v2/config v1.32.28
	github.com/aws/aws-sdk-go-v2/credentials v1.19.27
	github.com/aws/aws-sdk-go-v2/service/ecr v1.59.0
	github.com/aws/aws-sdk-go-v2/service/sts v1.44.0
	github.com/docker/cli v29.6.1+incompatible
	github.com/docker/docker-credential-helpers v0.9.8
	github.com/goccy/go-yaml v1.19.2
)

require (
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.18.30 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.4.30 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.7.30 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.4.31 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.13 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.13.30 // indirect
	github.com/aws/aws-sdk-go-v2/service/signin v1.3.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.32.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.37.0 // indirect
	github.com/aws/smithy-go v1.27.3 // indirect
	github.com/sirupsen/logrus v1.9.4 // indirect
	golang.org/x/sys v0.42.0 // indirect
	gotest.tools/v3 v3.5.2 // indirect
)

replace gopkg.in/yaml.v3 => go.yaml.in/yaml/v3 v3.0.4
