# A cli tool to create/scale/delete k8s clusters and deploy manifest files.
currently it supports GKE, but it is designed in a way that adding more providers should be easy.
## Build
The project uses [go modules](https://github.com/golang/go/wiki/Modules) so it requires go with support for modules.

```
go build cmd/prombench/*.go 
// reads go.mod from the project root and downloads all dependancies.
```

## Usage
```
./prombench -h  // Usage and examples.
```