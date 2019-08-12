module github.com/prometheus/prombench

go 1.12

require (
	cloud.google.com/go v0.39.0
	github.com/alecthomas/template v0.0.0-20160405071501-a0175ee3bccc // indirect
	github.com/alecthomas/units v0.0.0-20151022065526-2efee857e7cf // indirect
	github.com/gogo/protobuf v1.2.1 // indirect
	github.com/google/gofuzz v1.0.0 // indirect
	github.com/googleapis/gnostic v0.2.0 // indirect
	github.com/imdario/mergo v0.3.7 // indirect
	github.com/json-iterator/go v1.1.6 // indirect
	github.com/kr/pretty v0.1.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.1 // indirect
	github.com/pkg/errors v0.8.1
	github.com/prometheus/prombench/pkg/provider v0.0.0-00010101000000-000000000000 // indirect
	github.com/prometheus/prombench/pkg/provider/gke v0.0.0-00010101000000-000000000000
	github.com/prometheus/prombench/pkg/provider/k8s v0.0.0-00010101000000-000000000000 // indirect
	github.com/spf13/pflag v1.0.3 // indirect
	google.golang.org/api v0.5.0
	google.golang.org/genproto v0.0.0-20190516172635-bb713bdc0e52
	google.golang.org/grpc v1.19.0
	gopkg.in/alecthomas/kingpin.v2 v2.2.6
	gopkg.in/check.v1 v1.0.0-20180628173108-788fd7840127 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/yaml.v2 v2.2.2
	k8s.io/api v0.0.0-20190409021203-6e4e0e4f393b
	k8s.io/apiextensions-apiserver v0.0.0-20190409022649-727a075fdec8
	k8s.io/apimachinery v0.0.0-20190404173353-6a84e37a896d
	k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible
	k8s.io/utils v0.0.0-20190506122338-8fab8cb257d5 // indirect
	sigs.k8s.io/yaml v1.1.0 // indirect
)

replace (
	github.com/prometheus/prombench/pkg/provider => ../../pkg/provider
	github.com/prometheus/prombench/pkg/provider/gke => ../../pkg/provider/gke
	github.com/prometheus/prombench/pkg/provider/k8s => ../../pkg/provider/k8s
)
