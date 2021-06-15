// +build k8srequired

package templates

// AppOperatorValues values required by app-operator-chart.
const AppOperatorValues = `
 provider:
   kind: aws
 registry:
   domain: quay.io
`
