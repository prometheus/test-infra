# A cli tool to create/scale/delete k8s clusters and deploy manifest files.
currently it supports GKE, but it is designed in a way that adding more providers should be easy.
## Build
The project uses [vgo](https://github.com/golang/vgo) without any vendoring.
```
go get -u golang.org/x/vgo
vgo build -o prombench cmd/prombench/main.go 
// reads go.mod from the project root and downloads all dependancies.
```

## Usage
```
gke -h  // Usage and examples.
```