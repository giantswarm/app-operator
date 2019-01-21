package kubeconfig

import (
	"context"

	"github.com/giantswarm/apiextensions/pkg/clientset/versioned"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// Config represents the configuration used to create a new kubeconfig library instance.
type Config struct {
	Logger    micrologger.Logger
	K8sClient kubernetes.Interface
}

// KubeConfig provides functionality for connecting to tenant clusters based on the specified secret information.
type KubeConfig struct {
	logger    micrologger.Logger
	k8sClient kubernetes.Interface
}

// New creates a new KubeConfig service.
func New(config Config) (*KubeConfig, error) {
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}
	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sClient must not be empty", config)
	}

	g := &KubeConfig{
		logger:    config.Logger,
		k8sClient: config.K8sClient,
	}

	return g, nil
}

// NewG8sClientFromSecret returns a generated clientset based on the kubeconfig stored in a secret.
func (k KubeConfig) NewG8sClientFromSecret(ctx context.Context, secretName, secretNamespace string) (versioned.Interface, error) {
	kubeConfig, err := k.getKubeConfigFromSecret(ctx, secretName, secretNamespace)
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

// NewK8sClientFromSecret returns a Kubernetes clientset based on the kubeconfig stored in a secret.
func (k KubeConfig) NewK8sClientFromSecret(ctx context.Context, secretName, secretNamespace string) (kubernetes.Interface, error) {
	kubeConfig, err := k.getKubeConfigFromSecret(ctx, secretName, secretNamespace)
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
