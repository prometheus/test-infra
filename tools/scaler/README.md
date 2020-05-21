# A cli tool to scale k8s deployments from within a k8s cluster.

This tool uses [k8s provider](../../pkg/provider/k8s) to scale a deployment up and down periodically, from within a k8s cluster.

## RBAC Roles
The container running this tool should have the following RBAC configuration.
```
kind: Role
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: scaler
  namespace: prombench
rules:
- apiGroups: ["apps"]   #apiVersion of deployment being scaled
  resources:
  - deployments
  verbs: ["get", "list", "update"]
```


## Usage
```
// (Note: These commands should be executed inside a k8s container)
./scaler -h  // Usage and examples.

Sample Output of ./scaler help scale :

usage: scaler scale --file=FILE [<flags>] <max> <min> <interval>

Scale a Kubernetes deployment object periodically up and down.
ex: ./scaler scale -v NAMESPACE:scale -f fake-webserver.yaml 20 1 15m

Flags:
  -h, --help           Show context-sensitive help (also try --help-long and --help-man).
  -f, --file=FILE ...  yaml file or folder that describes the parameters for the deployment.
  -v, --vars=VARS ...  When provided it will substitute the token holders in the yaml file. Follows the standard golang template formating - {{ .hashStable }}.

Args:
  <max>       Number of Replicas to scale up.
  <min>       Number of Replicas to scale down.
  <interval>  Time to wait before changing the number of replicas.
```

### Building Docker Image
```
docker build -t prominfra/scaler:master .
```
