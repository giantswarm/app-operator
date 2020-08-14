// Package label contains common Kubernetes object labels. These are defined in
// https://github.com/giantswarm/fmt/blob/master/kubernetes/annotations_and_labels.md.
package label

const (
	// App label is used to identify Kubernetes resources.
	App = "app"

	// AppKubernetesVersion label is used to identify the version of Kubernetes
	// resources.
	AppKubernetesVersion = "app.kubernetes.io/version"
)
