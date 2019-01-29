package kubeconfig

import (
	"context"

	"github.com/giantswarm/apiextensions/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/apiextensions/pkg/clientset/versioned"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// Config represents the configuration used to create a new kubeconfig library
// instance.
type Config struct {
	G8sClient versioned.Interface
	Logger    micrologger.Logger
	K8sClient kubernetes.Interface
}

// KubeConfig provides functionality for connecting to remote clusters based on
// the specified kubeconfig.
type KubeConfig struct {
	g8sClient versioned.Interface
	logger    micrologger.Logger
	k8sClient kubernetes.Interface
}

// New creates a new KubeConfig service.
func New(config Config) (*KubeConfig, error) {
	if config.G8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.G8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}
	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sClient must not be empty", config)
	}

	g := &KubeConfig{
		g8sClient: config.G8sClient,
		logger:    config.Logger,
		k8sClient: config.K8sClient,
	}

	return g, nil
}

// NewG8sClientForApp returns a generated clientset for the cluster configured
// in the kubeconfig section of the app CR. If this is empty a clientset for
// the current cluster is returned.
func (k KubeConfig) NewG8sClientForApp(ctx context.Context, app v1alpha1.App) (versioned.Interface, error) {
	secretName := secretName(app)

	// KubeConfig is not configured so connect to current cluster.
	if secretName == "" {
		return k.g8sClient, nil
	}

	kubeConfig, err := k.getKubeConfigFromSecret(ctx, secretName, secretNamespace(app))
	if err != nil {
		return nil, err
	}

	restConfig, err := clientcmd.RESTConfigFromKubeConfig(kubeConfig)
	if err != nil {
		return nil, err
	}

	client, err := versioned.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}
	return client, nil
}

// NewK8sClientForApp returns a Kubernetes clientset for the cluster configured
// in the kubeconfig section of the app CR. If this is empty a clientset for
// the current cluster is returned.
func (k KubeConfig) NewK8sClientForApp(ctx context.Context, app v1alpha1.App) (kubernetes.Interface, error) {
	secretName := secretName(app)

	// KubeConfig is not configured so connect to current cluster.
	if secretName == "" {
		return k.k8sClient, nil
	}

	kubeConfig, err := k.getKubeConfigFromSecret(ctx, secretName, secretNamespace(app))
	if err != nil {
		return nil, err
	}

	restConfig, err := clientcmd.RESTConfigFromKubeConfig(kubeConfig)
	if err != nil {
		return nil, err
	}

	client, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}
	return client, nil
}

// getKubeConfigFromSecret returns KubeConfig bytes based on the specified secret information.
func (k KubeConfig) getKubeConfigFromSecret(ctx context.Context, secretName, secretNamespace string) ([]byte, error) {
	secret, err := k.k8sClient.CoreV1().Secrets(secretNamespace).Get(secretName, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		return nil, err
	} else if _, isStatus := err.(*errors.StatusError); isStatus {
		return nil, err
	} else if err != nil {
		return nil, err
	}
	if bytes, ok := secret.Data["kubeConfig"]; ok {
		return bytes, nil
	} else {
		return nil, notFoundError
	}
}
