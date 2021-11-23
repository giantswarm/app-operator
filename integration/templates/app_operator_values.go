//go:build k8srequired
// +build k8srequired

package templates

// AppOperatorValues values required by app-operator-chart.
const AppOperatorValues = `
  registry:
    domain: quay.io
`
