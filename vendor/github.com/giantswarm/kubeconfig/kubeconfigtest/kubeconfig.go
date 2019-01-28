package kubeconfigtest

import (
	"context"

	"github.com/giantswarm/apiextensions/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/apiextensions/pkg/clientset/versioned"
	"github.com/giantswarm/kubeconfig"
	"k8s.io/client-go/kubernetes"
)

type Config struct {
	G8sClient             versioned.Interface
	G8sClientFromAppError error
	K8sClient             kubernetes.Interface
	K8sClientFromAppError error
}

type KubeConfig struct {
	g8sClient             versioned.Interface
	g8sClientFromAppError error
	k8sClient             kubernetes.Interface
	k8sClientFromAppError error
}

func New(config Config) kubeconfig.Interface {
	k := &KubeConfig{
		g8sClient:             config.G8sClient,
		g8sClientFromAppError: config.G8sClientFromAppError,
		k8sClient:             config.K8sClient,
		k8sClientFromAppError: config.K8sClientFromAppError,
	}

	return k
}

func (k *KubeConfig) NewG8sClientForApp(ctx context.Context, app v1alpha1.App) (versioned.Interface, error) {
	if k.g8sClientFromAppError != nil {
		return nil, k.g8sClientFromAppError
	}

	return k.g8sClient, nil
}

func (k *KubeConfig) NewK8sClientForApp(ctx context.Context, app v1alpha1.App) (kubernetes.Interface, error) {
	if k.k8sClientFromAppError != nil {
		return nil, k.k8sClientFromAppError
	}

	return k.k8sClient, nil
}
