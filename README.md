#TO_DO 

allow ovverrides from cli <br/>
    - MachineType: {{ if .MachineType }} {{ .MachineType }} {{ else }} n1-standard-1 {{ end }}
prombench cluster create --ovverides=MachineType=....,ram=....

can we simplify update if exists , create if doesn't

test what happens with template variabls - var exists in resorce file , but not passed to cli , var passed to cli but doesn't exist in the file.

Service update doesn't work!
use more sane logging that shows the line in the code of the log  and show info logs only when debug mode is enabled.

# A Kubernetes cluster preconfigured for testing Prometheus


## Build
The project uses [vgo](https://github.com/golang/vgo) without any vendoring.
```
go get -u golang.org/x/vgo
cd cmd/prombench
vgo build // This will read go.mod from project root and download all dependancies.
```

## Pre-requisites
1. Create a new GKE project.
2. Create a project service account and save the json file in `cmd/prombench`.

## Usage
`gke -h`  - for usage and examples
