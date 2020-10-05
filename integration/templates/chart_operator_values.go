// +build k8srequired

package templates

// ChartOperatorValues values required by chart-operator-chart.
const ChartOperatorValues = `
clusterDNSIP: 10.96.0.10
externalDNSIP: 8.8.8.8
image:
  registry: "quay.io"
tiller:
  namespace: "giantswarm"
`
