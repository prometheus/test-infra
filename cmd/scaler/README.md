# A cli tool to scale k8s deployments from within a k8s cluster. 

This tool uses [k8s provider](../../pkg/provider/k8s) to scale a deployment up and down periodically, from within a k8s cluster.

## Build
The project uses [go modules](https://github.com/golang/go/wiki/Modules) so it requires go with support for modules.

```
go build scaler.go
// reads go.mod from the project root and downloads all dependancies.
```

## Usage
```
// (Note: These commands should be executed inside a k8s container)
./scaler -h  // Usage and examples. 

Sample Output of ./scaler --help-long :
usage: scaler [<flags>] <command> [<args> ...]

The Prombench-Scaler tool

Flags:
  -h, --help  Show context-sensitive help (also try --help-long and --help-man).

Commands:
  help [<command>...]
    Show help.


  scale --file=FILE [<flags>] <max> <min> <interval>
    Scale a Kubernetes deployment object periodically up and down. ex: ./scaler scale -v NAMESPACE:scale -f fake-webserver.yaml 20 1 15m

    -f, --file=FILE ...  yaml file or folder that describes the parameters for the deployment.
    -v, --vars=VARS ...  When provided it will substitute the token holders in the yaml file. Follows the standard golang template formating - {{ .hashStable }}.
```