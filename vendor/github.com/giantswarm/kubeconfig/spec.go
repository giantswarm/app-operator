package kubeconfig

import (
	"context"

	"github.com/giantswarm/apiextensions/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/apiextensions/pkg/clientset/versioned"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type Interface interface {
	// NewG8sClientForApp returns a generated clientset for the cluster configured
	// in the kubeconfig section of the app CR. If this is empty a clientset for
	// the current cluster is returned.
	NewG8sClientForApp(ctx context.Context, app v1alpha1.App) (versioned.Interface, error)

	// NewK8sClientForApp returns a Kubernetes clientset for the cluster configured
	// in the kubeconfig section of the app CR. If this is empty a clientset for
	// the current cluster is returned.
	NewK8sClientForApp(ctx context.Context, app v1alpha1.App) (kubernetes.Interface, error)

	// NewRESTConfigForApp returns a Kubernetes REST Config for the cluster configured
	// in the kubeconfig section of the app CR.
	NewRESTConfigForApp(ctx context.Context, app v1alpha1.App) (*rest.Config, error)
}
