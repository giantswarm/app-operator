// +build k8srequired

package templates

// ChartOperatorValues values required by chart-operator-chart.
const ChartOperatorValues = `clusterDNSIP: 10.96.0.10
e2e: true

Installation:
  V1:
    Helm:
      HTTP:
        ClientTimeout: "30s"
      Kubernetes:
        WaitTimeout: "180s"
    Registry:
      Domain: "quay.io"`
