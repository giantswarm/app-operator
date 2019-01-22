package kubeconfig

import (
	"context"

	"github.com/giantswarm/apiextensions/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/apiextensions/pkg/clientset/versioned"
	kubeconfiglib "github.com/giantswarm/kubeconfig"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"k8s.io/client-go/kubernetes"

	"github.com/giantswarm/app-operator/service/controller/app/v1/key"
)

// Config represents the configuration used to create a new kubeconfig service.
type Config struct {
	G8sClient versioned.Interface
	K8sClient kubernetes.Interface
	Logger    micrologger.Logger
}

// KubeConfig service provides primitives for connecting to the Kubernetes
// cluster configured in the kubeconfig section of the app CR.
type KubeConfig struct {
	g8sClient  versioned.Interface
	k8sClient  kubernetes.Interface
	kubeConfig kubeconfiglib.Interface
	logger     micrologger.Logger
}

// New creates a new configured kubeconfig service.
func New(config Config) (*KubeConfig, error) {
	var err error

	if config.G8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.G8sClient must not be empty", config)
	}
	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	var kc kubeconfiglib.Interface
	{
		c := kubeconfiglib.Config{
			Logger:    config.Logger,
			K8sClient: config.K8sClient,
		}

		kc, err = kubeconfiglib.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	k := &KubeConfig{
		g8sClient:  config.G8sClient,
		k8sClient:  config.K8sClient,
		kubeConfig: kc,
		logger:     config.Logger,
	}

	return k, nil
}

// NewG8sClientForApp returns a generated clientset for the cluster configured
// in the kubeconfig section of the app CR. If this is empty a clientset for
// the current cluster is returned.
func (k KubeConfig) NewG8sClientForApp(ctx context.Context, customResource v1alpha1.App) (versioned.Interface, error) {
	secretName := key.KubeConfigSecretName(customResource)

	// KubeConfig is not configured so connect to current cluster.
	if secretName == "" {
		return k.g8sClient, nil
	}

	secretNamespace := key.KubeConfigSecretNamespace(customResource)
	k8sClient, err := k.kubeConfig.NewG8sClientFromSecret(ctx, secretName, secretNamespace)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return k8sClient, nil
}

// NewK8sClientForApp returns a Kubernetes clientset for the cluster configured
// in the kubeconfig section of the app CR. If this is empty a clientset for
// the current cluster is returned.
func (k KubeConfig) NewK8sClientForApp(ctx context.Context, customResource v1alpha1.App) (kubernetes.Interface, error) {
	secretName := key.KubeConfigSecretName(customResource)

	// KubeConfig is not configured so connect to current cluster.
	if secretName == "" {
		return k.k8sClient, nil
	}

	secretNamespace := key.KubeConfigSecretNamespace(customResource)
	k8sClient, err := k.kubeConfig.NewK8sClientFromSecret(ctx, secretName, secretNamespace)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return k8sClient, nil
}
