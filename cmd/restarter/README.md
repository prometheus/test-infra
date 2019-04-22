# A cli tool to restart k8s deployments from within a k8s cluster. 

This tool uses [k8s provider](../../pkg/provider/k8s) to restart a deployment

## Build
The project uses [go modules](https://github.com/golang/go/wiki/Modules) so it requires go with support for modules.

```
$ make build
```

## RBAC Roles
The container running this tool should have the following RBAC configuration.
```
kind: Role
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: loadgen-restarter
  namespace: prombench-{{ .PR_NUMBER }}
rules:
- apiGroups: [""]
  resources:
  - pods
  - pods/exec
  verbs: ["list", "get"]
```

## Usage
```
// (Note: These commands should be executed inside a k8s container)
./restarter -h  // Usage and examples. 

Restart a Kubernetes deployment object
ex: ./restarter restart -v PR_NUMBER:prometheus-5500

Flags:
  -h, --help           Show context-sensitive help (also try --help-long and --help-man).
  -v, --vars=VARS ...  When provided it will substitute the token holders in the yaml file. Follows the standard golang template formating - {{ .hashStable }}.
```