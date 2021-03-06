[![CircleCI](https://circleci.com/gh/giantswarm/app-operator.svg?style=shield)](https://circleci.com/gh/giantswarm/app-operator) 

# app-operator

The app-operator manages apps in Kubernetes clusters. It is implemented
using [operatorkit]. 

## Branches

- `master`
    - Latest version using Helm 3.
- `helm2`
    - Legacy support for Helm 2.

## app CR

The operator deploys charts hosted in a Helm repository. The app CRs are
used to generate chart CRs managed by [chart-operator] which is our agent
for automating deployments with Helm.

### Example app CR

```yaml
apiVersion: application.giantswarm.io/v1alpha1
kind: App
metadata:
  creationTimestamp: null
  labels:
    app-operator.giantswarm.io/version: 1.0.0
  name: prometheus
  namespace: default
spec:
  catalog: my-playground-catalog
  config:
    configMap:
      name: f2def-cluster-values
      namespace: f2def
    secret:
      name: f2def-cluster-values
      namespace: f2def
  kubeConfig:
    context:
      name: f2def
    inCluster: false
    secret:
      name: f2def-kubeconfig
      namespace: f2def
  name: prometheus
  namespace: monitoring
  userConfig:
    configMap:
      name: prometheus-user-values
      namespace: f2def
    secret:
      name: prometheus-user-values
      namespace: f2def
  version: 1.0.1
```

## Getting Project

Clone the git repository: https://github.com/giantswarm/app-operator.git

### How to build

Build it using the standard `go build` command.

```
go build github.com/giantswarm/app-operator
```

## Contact

- Mailing list: [giantswarm](https://groups.google.com/forum/!forum/giantswarm)
- IRC: #[giantswarm](irc://irc.freenode.org:6667/#giantswarm) on freenode.org
- Bugs: [issues](https://github.com/giantswarm/app-operator/issues)

## Contributing & Reporting Bugs

See [CONTRIBUTING](CONTRIBUTING.md) for details on submitting patches, the
contribution workflow as well as reporting bugs.

## License

app-operator is under the Apache 2.0 license. See the [LICENSE](LICENSE) file for
details.



[chart-operator]: https://github.com/giantswarm/chart-operator
[helm]: https://github.com/helm/helm
[operatorkit]: https://github.com/giantswarm/operatorkit
