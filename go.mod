module github.com/prometheus/test-infra

go 1.12

require (
	cloud.google.com/go v0.56.0
	github.com/aws/aws-sdk-go v1.34.5
	github.com/go-git/go-git-fixtures/v4 v4.0.1
	github.com/go-git/go-git/v5 v5.1.0
	github.com/google/go-github/v29 v29.0.3
	github.com/googleapis/gnostic v0.2.0 // indirect
	github.com/oklog/run v1.1.0
	github.com/pkg/errors v0.9.1
	github.com/prometheus/alertmanager v0.21.0
	github.com/prometheus/client_golang v1.6.0
	github.com/prometheus/common v0.10.0
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d
	golang.org/x/perf v0.0.0-20200318175901-9c9101da8316
	google.golang.org/api v0.27.0
	google.golang.org/genproto v0.0.0-20200331122359-1ee6d9798940
	google.golang.org/grpc v1.28.0
	gopkg.in/alecthomas/kingpin.v2 v2.2.6
	gopkg.in/yaml.v2 v2.3.0
	k8s.io/api v0.18.4
	k8s.io/apiextensions-apiserver v0.18.4
	k8s.io/apimachinery v0.18.4
	k8s.io/client-go v0.18.4
	sigs.k8s.io/aws-iam-authenticator v0.5.1
	sigs.k8s.io/kind v0.8.1
)
