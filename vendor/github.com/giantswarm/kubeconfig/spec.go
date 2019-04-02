package kubeconfig

import (
	"context"

	"github.com/giantswarm/apiextensions/pkg/apis/application/v1alpha1"
	"k8s.io/client-go/rest"
)

type Interface interface {
	// NewRESTConfigForApp returns a Kubernetes REST Config for the cluster configured
	// in the kubeconfig section of the app CR.
	NewRESTConfigForApp(ctx context.Context, app v1alpha1.App) (*rest.Config, error)
}
