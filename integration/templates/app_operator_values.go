//go:build k8srequired
// +build k8srequired

package templates

// AppOperatorVintageValues values required by app-operator-chart.
const AppOperatorVintageValues = `
 provider:
   kind: aws
 registry:
   domain: quay.io
`

// AppOperatorCAPIValues values required by unique app-operator-chart.
const AppOperatorCAPIValues = `
 app:
   helmControllerBackend: true
 provider:
   kind: aws
 registry:
   domain: quay.io
`

// AppOperatorCAPIValues values required by unique app-operator-chart.
const AppOperatorCAPIWCNewValues = `
 app:
   helmControllerBackend: true
   workloadClusterID: kind
   watchNamespace: org-test
 provider:
   kind: aws
 registry:
   domain: quay.io
`

// AppOperatorCAPIValues values required by unique app-operator-chart.
const AppOperatorCAPIWCOldValues = `
 app:
   workloadClusterID: kind
   watchNamespace: org-test
 provider:
   kind: aws
 registry:
   domain: quay.io
`
