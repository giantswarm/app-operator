// +build k8srequired

package templates

// ChartOperatorValues values required by chart-operator-chart.
const ChartOperatorValues = `clusterDNSIP: 10.96.0.10
e2e: true

helm:
  http:
    clientTimeout: "30s"
  kubernetes:
    waitTimeout: "180s"

registry:
  domain: "quay.io"`
