//go:build k8srequired
// +build k8srequired

package templates

// AppOperatorValues values required by app-operator-chart.
const AppOperatorValues = `
  chart:
    webhook:
      authToken: auth-token
      baseURL: g8s.gauss.eu-west-1.aws.gigantic.io
  provider:
    kind: aws
  registry:
    domain: quay.io
`
