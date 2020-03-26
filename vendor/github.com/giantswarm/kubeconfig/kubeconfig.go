package kubeconfig

import (
	"context"
	"encoding/base64"
	"fmt"
	yaml "gopkg.in/yaml.v2"

	"github.com/giantswarm/apiextensions/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// KubeConfig provides functionality for connecting to remote clusters based on
// the specified kubeconfig.
type KubeConfig struct {
	logger    micrologger.Logger
	k8sClient kubernetes.Interface
}

// New creates a new KubeConfig service.
func New(config Config) (*KubeConfig, error) {
	err := config.Validate()
	if err != nil {
		return nil, microerror.Mask(err)
	}

	g := &KubeConfig{
		logger:    config.Logger,
		k8sClient: config.K8sClient,
	}

	return g, nil
}

// NewRESTConfigForApp returns a Kubernetes REST config for the cluster
// configured in the kubeconfig section of the app CR.
func (k *KubeConfig) NewRESTConfigForApp(ctx context.Context, app v1alpha1.App) (*rest.Config, error) {
	if inCluster(app) {
		config, err := rest.InClusterConfig()
		if err != nil {
			return nil, microerror.Mask(err)
		}
		return config, nil
	}

	secretName := secretName(app)
	secretNamespace := secretNamespace(app)

	kubeConfig, err := k.getKubeConfigFromSecret(ctx, secretName, secretNamespace)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	restConfig, err := clientcmd.RESTConfigFromKubeConfig(kubeConfig)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	return restConfig, nil
}

// NewKubeConfigForRESTConfig returns a kubeConfig bytes for the given REST Config.
func NewKubeConfigForRESTConfig(ctx context.Context, config *rest.Config, clusterName, namespace string) ([]byte, error) {
	if config == nil {
		return nil, microerror.Maskf(executionFailedError, "config must not be empty")
	}
	if clusterName == "" {
		return nil, microerror.Maskf(executionFailedError, "clusterName must not be empty")
	}

	kubeConfig := KubeConfigValue{
		APIVersion: "v1",
		Kind:       "Config",
		Clusters: []KubeconfigNamedCluster{
			{
				Name: clusterName,
				Cluster: KubeconfigCluster{
					Server:                   config.Host,
					CertificateAuthorityData: base64.StdEncoding.EncodeToString(config.TLSClientConfig.CAData),
				},
			},
		},
		Contexts: []KubeconfigNamedContext{
			{
				Name: fmt.Sprintf("%s-context", clusterName),
				Context: KubeconfigContext{
					Cluster:   clusterName,
					Namespace: namespace,
					User:      fmt.Sprintf("%s-user", clusterName),
				},
			},
		},
		Users: []KubeconfigUser{
			{
				Name: fmt.Sprintf("%s-user", clusterName),
				User: KubeconfigUserKeyPair{
					ClientCertificateData: base64.StdEncoding.EncodeToString(config.CertData),
					ClientKeyData:         base64.StdEncoding.EncodeToString(config.KeyData),
				},
			},
		},
		CurrentContext: fmt.Sprintf("%s-context", clusterName),
	}

	bytes, err := yaml.Marshal(kubeConfig)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	return bytes, nil
}

// NewRESTConfigForKubeConfig returns a REST Config for the given KubeConfigValue.
func NewRESTConfigForKubeConfig(ctx context.Context, kubeConfig []byte) (*rest.Config, error) {
	restConfig, err := clientcmd.RESTConfigFromKubeConfig(kubeConfig)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	return restConfig, nil
}

// getKubeConfigFromSecret returns KubeConfig bytes based on the specified secret information.
func (k *KubeConfig) getKubeConfigFromSecret(ctx context.Context, secretName, secretNamespace string) ([]byte, error) {
	secret, err := k.k8sClient.CoreV1().Secrets(secretNamespace).Get(secretName, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		return nil, microerror.Maskf(notFoundError, "Secret %#q in Namespace %#q", secretName, secretNamespace)
	} else if _, isStatus := err.(*errors.StatusError); isStatus {
		return nil, microerror.Mask(err)
	} else if err != nil {
		return nil, microerror.Mask(err)
	}
	if bytes, ok := secret.Data["kubeConfig"]; ok {
		return bytes, nil
	} else {
		return nil, microerror.Maskf(notFoundError, "Secret %#q in Namespace %#q does not have kubeConfig key in its data", secretName, secretNamespace)
	}
}

func marshal(config *KubeConfigValue) ([]byte, error) {
	bytes, err := yaml.Marshal(config)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	return bytes, nil
}

func unmarshal(bytes []byte) (*KubeConfigValue, error) {
	var kubeConfig KubeConfigValue
	err := yaml.Unmarshal(bytes, &kubeConfig)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	return &kubeConfig, nil
}
