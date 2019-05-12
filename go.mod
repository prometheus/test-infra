module github.com/prometheus/prombench

go 1.12

require (
	cloud.google.com/go v0.38.0
	github.com/gogo/protobuf v1.2.1 // indirect
	github.com/golang/protobuf v1.3.1 // indirect
	github.com/google/gofuzz v1.0.0 // indirect
	github.com/googleapis/gnostic v0.2.0 // indirect
	github.com/gregjones/httpcache v0.0.0-20190212212710-3befbb6ad0cc // indirect
	github.com/hashicorp/golang-lru v0.5.1 // indirect
	github.com/imdario/mergo v0.3.7 // indirect
	github.com/json-iterator/go v1.1.6 // indirect
	github.com/kr/pretty v0.1.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.1 // indirect
	github.com/peterbourgon/diskv v2.0.1+incompatible // indirect
	github.com/pkg/errors v0.8.1
	github.com/prometheus/client_golang v0.9.3-0.20190127221311-3c4408c8b829
	github.com/spf13/pflag v1.0.3 // indirect
	github.com/stretchr/testify v1.3.0 // indirect
	golang.org/x/crypto v0.0.0-20190404164418-38d8ce5564a5 // indirect
	golang.org/x/net v0.0.0-20190403144856-b630fd6fe46b // indirect
	golang.org/x/text v0.3.2 // indirect
	google.golang.org/genproto v0.0.0-20190508193815-b515fa19cec8
	google.golang.org/grpc v1.20.1
	gopkg.in/alecthomas/kingpin.v2 v2.2.6
	gopkg.in/check.v1 v1.0.0-20180628173108-788fd7840127 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/yaml.v2 v2.2.2
	k8s.io/api v0.0.0-20190510023542-0cde1e56adb1
	k8s.io/apiextensions-apiserver v0.0.0-20190510104120-a79159fa8be4
	k8s.io/apimachinery v0.0.0-20190509063443-7d8f8feb49c5
	k8s.io/client-go v11.0.0+incompatible
	k8s.io/klog v0.3.0 // indirect
	sigs.k8s.io/yaml v1.1.0 // indirect
)

replace (
	k8s.io/api => k8s.io/api v0.0.0-20190222213804-5cb15d344471
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.0.0-20190221221350-bfb440be4b87
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20190221213512-86fb29eff628
	k8s.io/client-go => k8s.io/client-go v10.0.0+incompatible
)
