// +build k8srequired

package templates

// ChartOperatorValues values required by chart-operator-chart.
const ChartOperatorValues = `
clusterDNSIP: 10.96.0.10
<<<<<<< HEAD
=======
image:
  registry: "quay.io"
>>>>>>> master
tiller:
  namespace: "giantswarm"
`
