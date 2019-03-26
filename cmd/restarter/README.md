# A cli tool to restart k8s deployments from within a k8s cluster. 

This tool uses [k8s provider](../../pkg/provider/k8s) to restart a deployment

## Build
The project uses [go modules](https://github.com/golang/go/wiki/Modules) so it requires go with support for modules.

```
go build restarter.go
// reads go.mod from the project root and downloads all dependencies.
```

## RBAC Roles
The container running this tool should have the following RBAC configuration.
```
kind: Role
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: loadgen-restarter
  namespace: prombench
rules:
- apiGroups: ["apps"]   #apiVersion of deployment being restarted
  resources:
  - deployments
  verbs: ["get", "list", "update"]
```

## Usage
```
// (Note: These commands should be executed inside a k8s container)
./restarter -h  // Usage and examples. 

Sample Output of ./restarter help restart :

usage: restarter restart --file=FILE [<flags>]

Restart a Kubernetes deployment object
ex: ./restarter restart -v NAMESPACE:restart -f 3_prometheus_meta.yaml

Flags:
  -h, --help           Show context-sensitive help (also try --help-long and --help-man).
  -f, --file=FILE ...  yaml file or folder that describes the parameters for the deployment.
  -v, --vars=VARS ...  When provided it will substitute the token holders in the yaml file. Follows the standard golang template formating - {{ .hashStable }}.
```
