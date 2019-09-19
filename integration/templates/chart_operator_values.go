// +build k8srequired

package templates

// ChartOperatorValues values required by chart-operator-chart.
const ChartOperatorValues = `
clusterDNSIP: 10.96.0.10
image:
  registry: "quay.io"
tiller:
  namespace: "giantswarm"
`
