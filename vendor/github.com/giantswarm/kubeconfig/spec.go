package kubeconfig

import (
	"context"

	"github.com/giantswarm/apiextensions/pkg/clientset/versioned"
	"k8s.io/client-go/kubernetes"
)

type Interface interface {
	// NewG8sClientFromSecret returns a generated clientset based on the kubeconfig stored in a secret.
	NewG8sClientFromSecret(ctx context.Context, secretName, secretNamespace string) (versioned.Interface, error)
	// NewK8sClientFromSecret returns a Kubernetes clientset based on the kubeconfig stored in a secret.
	NewK8sClientFromSecret(ctx context.Context, secretName, secretNamespace string) (kubernetes.Interface, error)
}
