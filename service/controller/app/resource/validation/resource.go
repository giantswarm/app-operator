package validation

import (
	"context"

	"github.com/giantswarm/apiextensions/v3/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/apiextensions/v3/pkg/clientset/versioned"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/giantswarm/app-operator/v2/service/controller/key"
)

const (
	Name = "validation"

	resourceNotFoundTemplate        = "%s %#q in namespace %#q not found"
	namespaceNotFoundReasonTemplate = "namespace is not specified for %s %#q"
)

// Config represents the configuration used to create a new chartstatus resource.
type Config struct {
	G8sClient versioned.Interface
	K8sClient kubernetes.Interface
	Logger    micrologger.Logger
}

// Resource implements the chartstatus resource.
type Resource struct {
	g8sClient versioned.Interface
	k8sClient kubernetes.Interface
	logger    micrologger.Logger
}

func New(config Config) (*Resource, error) {
	if config.G8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.G8sClient must not be empty", config)
	}
	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	r := &Resource{
		// Dependencies.
		g8sClient: config.G8sClient,
		k8sClient: config.K8sClient,
		logger:    config.Logger,
	}

	return r, nil
}

func (r Resource) Name() string {
	return Name
}

func (r *Resource) validateApp(ctx context.Context, cr v1alpha1.App) error {
	if key.AppConfigMapName(cr) != "" {
		ns := key.AppConfigMapNamespace(cr)
		if ns == "" {
			return microerror.Maskf(validationError, namespaceNotFoundReasonTemplate, "configmap", key.AppConfigMapName(cr))
		}

		_, err := r.k8sClient.CoreV1().ConfigMaps(ns).Get(ctx, key.AppConfigMapName(cr), metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			return microerror.Maskf(validationError, resourceNotFoundTemplate, "configmap", key.AppConfigMapName(cr), ns)
		} else if err != nil {
			return microerror.Mask(err)
		}
	}

	if key.AppSecretName(cr) != "" {
		ns := key.AppSecretNamespace(cr)
		if ns == "" {
			return microerror.Maskf(validationError, namespaceNotFoundReasonTemplate, "secret", key.AppSecretName(cr))
		}

		_, err := r.k8sClient.CoreV1().Secrets(ns).Get(ctx, key.AppSecretName(cr), metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			return microerror.Maskf(validationError, resourceNotFoundTemplate, "secret", key.AppSecretName(cr), ns)
		} else if err != nil {
			return microerror.Mask(err)
		}
	}

	if key.UserConfigMapName(cr) != "" {
		ns := key.UserConfigMapNamespace(cr)
		if ns == "" {
			return microerror.Maskf(validationError, namespaceNotFoundReasonTemplate, "configmap", key.UserConfigMapName(cr))
		}

		_, err := r.k8sClient.CoreV1().ConfigMaps(ns).Get(ctx, key.UserConfigMapName(cr), metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			return microerror.Maskf(validationError, resourceNotFoundTemplate, "configmap", key.UserConfigMapName(cr), ns)
		} else if err != nil {
			return microerror.Mask(err)
		}
	}

	if key.UserSecretName(cr) != "" {
		ns := key.UserSecretNamespace(cr)
		if ns == "" {
			return microerror.Maskf(validationError, namespaceNotFoundReasonTemplate, "secret", key.UserSecretName(cr))
		}

		_, err := r.k8sClient.CoreV1().Secrets(key.UserSecretNamespace(cr)).Get(ctx, key.UserSecretName(cr), metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			return microerror.Maskf(validationError, resourceNotFoundTemplate, "secret", key.UserSecretName(cr), ns)
		} else if err != nil {
			return microerror.Mask(err)
		}
	}

	if !key.InCluster(cr) {
		ns := key.KubecConfigSecretNamespace(cr)
		if ns == "" {
			return microerror.Maskf(validationError, namespaceNotFoundReasonTemplate, "kubeconfig secret", key.KubecConfigSecretName(cr))
		}

		_, err := r.k8sClient.CoreV1().Secrets(key.KubecConfigSecretNamespace(cr)).Get(ctx, key.KubecConfigSecretName(cr), metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			return microerror.Maskf(validationError, resourceNotFoundTemplate, "kubeconfig secret", key.KubecConfigSecretName(cr), ns)
		} else if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}
