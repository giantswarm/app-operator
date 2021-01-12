// +build k8srequired

package templates

// ChartOperatorValues values required by chart-operator-chart.
const ChartOperatorValues = `Installation:
  V1:
    Helm:
      HTTP:
        ClientTimeout: "30s"
      Kubernetes:
        WaitTimeout: "180s"
    Registry:
      Domain: "quay.io"`
