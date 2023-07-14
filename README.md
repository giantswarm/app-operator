[![CircleCI](https://dl.circleci.com/status-badge/img/gh/giantswarm/app-operator/tree/master.svg?style=svg)](https://dl.circleci.com/status-badge/redirect/gh/giantswarm/app-operator/tree/master)

# app-operator

The app-operator manages apps in Kubernetes clusters. It is implemented
using [operatorkit].

## Important

Upon releasing a new version of the project, remember to reference it in the [Cluster Apps Operator Helm Chart](https://github.com/giantswarm/cluster-apps-operator/blob/28d9692bdff1e1f8a95b948cb91f593a5ec97536/helm/cluster-apps-operator/values.yaml#L3).

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

### Updating Chart CRD

- For workload clusters app-operator manages the chart CRD.
- When changes are made in [apiextensions-application](https://github.com/giantswarm/apiextensions-application)
they need to be synced here.

```sh
$ make sync-chart-crd
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
