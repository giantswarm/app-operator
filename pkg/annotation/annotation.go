// Package label contains common Kubernetes object labels. These are defined in
// https://github.com/giantswarm/fmt/blob/master/kubernetes/annotations_and_labels.md.
package annotation

const (
	// AppNamespace annotation is used by the chart status watcher to find the
	// app CR for chart CRs which are always in the giantswarm namespace.
	AppNamespace = "chart-operator.giantswarm.io/app-namespace"
)
