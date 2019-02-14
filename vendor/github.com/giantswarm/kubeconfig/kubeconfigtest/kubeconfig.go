package kubeconfigtest

import (
	"context"

	"github.com/giantswarm/apiextensions/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/kubeconfig"
	"k8s.io/client-go/rest"
)

type Config struct {
	RestConfig             rest.Config
	RestConfigFromAppError error
}

type KubeConfig struct {
	restConfig             rest.Config
	restConfigFromAppError error
}

func New(config Config) kubeconfig.Interface {
	k := &KubeConfig{
		restConfig:             config.RestConfig,
		restConfigFromAppError: config.RestConfigFromAppError,
	}

	return k
}

func (k *KubeConfig) NewRESTConfigForApp(ctx context.Context, app v1alpha1.App) (*rest.Config, error) {
	if k.restConfigFromAppError != nil {
		return nil, k.restConfigFromAppError
	}

	return &k.restConfig, nil
}
