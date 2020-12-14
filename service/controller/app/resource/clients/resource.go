package clients

import (
	"context"
	"time"

	"github.com/giantswarm/apiextensions/v3/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/app/v4/pkg/key"
	"github.com/giantswarm/errors/tenant"
	"github.com/giantswarm/helmclient/v4/pkg/helmclient"
	"github.com/giantswarm/k8sclient/v5/pkg/k8sclient"
	"github.com/giantswarm/kubeconfig/v4"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/afero"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"

	"github.com/giantswarm/app-operator/v2/service/controller/app/controllercontext"
)

const (
	// Name is the identifier of the resource.
	Name = "clients"
)

// Config represents the configuration used to create a new clients resource.
type Config struct {
	// Dependencies.
	Fs         afero.Fs
	HelmClient helmclient.Interface
	K8sClient  k8sclient.Interface
	Logger     micrologger.Logger

	// Settings.
	HTTPClientTimeout time.Duration
}

// Resource implements the clients resource.
type Resource struct {
	// Dependencies.
	fs         afero.Fs
	helmClient helmclient.Interface
	k8sClient  k8sclient.Interface
	logger     micrologger.Logger

	// Settings.
	httpClientTimeout time.Duration
}

// New creates a new configured clients resource.
func New(config Config) (*Resource, error) {
	if config.Fs == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Fs must not be empty", config)
	}
	if config.HelmClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.HelmClient must not be empty", config)
	}
	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	if config.HTTPClientTimeout == 0 {
		return nil, microerror.Maskf(invalidConfigError, "%T.HTTPClientTimeout must not be empty", config)
	}

	r := &Resource{
		// Dependencies.
		fs:         config.Fs,
		helmClient: config.HelmClient,
		k8sClient:  config.K8sClient,
		logger:     config.Logger,

		// Settings
		httpClientTimeout: config.HTTPClientTimeout,
	}

	return r, nil
}

func (*Resource) Name() string {
	return Name
}

// addClientsToContext adds g8s and k8s clients based on the kubeconfig
// settings for the app CR.
func (r *Resource) addClientsToContext(ctx context.Context, cr v1alpha1.App) error {
	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	if cc.Status.ClusterStatus.IsDeleting {
		return nil
	}

	// App CR uses inCluster so we can reuse the existing clients.
	if key.InCluster(cr) {
		cc.Clients = controllercontext.Clients{
			Ctrl: r.k8sClient.CtrlClient(),
			Helm: r.helmClient,
		}

		return nil
	}

	var kubeConfig kubeconfig.Interface
	{
		c := kubeconfig.Config{
			K8sClient: r.k8sClient.K8sClient(),
			Logger:    r.logger,
		}

		kubeConfig, err = kubeconfig.New(c)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	var restConfig *rest.Config
	{
		restConfig, err = kubeConfig.NewRESTConfigForApp(ctx, key.KubeConfigSecretName(cr), key.KubeConfigSecretNamespace(cr))
		if kubeconfig.IsNotFoundError(err) {
			// Set status so we don't try to connect to the tenant cluster
			// again in this reconciliation loop.
			cc.Status.ClusterStatus.IsUnavailable = true

			r.logger.Debugf(ctx, "kubeconfig secret not found")
			r.logger.Debugf(ctx, "canceling resource")
			return nil

		} else if err != nil {
			return microerror.Mask(err)
		}
	}

	var ctrlClient client.Client
	{
		var schemeConfig = scheme.Scheme

		// Extend the global client-go scheme which is used by all the tools under
		// the hood. The scheme is required for the controller-runtime controller to
		// be able to watch for runtime objects of a certain type.
		appSchemeBuilder := runtime.SchemeBuilder(schemeBuilder{
			v1alpha1.AddToScheme,
			apiextensionsv1.AddToScheme,
		})
		err = appSchemeBuilder.AddToScheme(schemeConfig)
		if err != nil {
			return microerror.Mask(err)
		}

		// Configure a dynamic rest mapper to the controller client so it can work
		// with runtime objects of arbitrary types. Note that this is the default
		// for controller clients created by controller-runtime managers.
		// Anticipating a rather uncertain future and more breaking changes to come
		// we want to separate client and manager. Thus we configure the client here
		// properly on our own instead of relying on the manager to provide a
		// client, which might change in the future.
		mapper, err := apiutil.NewDynamicRESTMapper(rest.CopyConfig(restConfig))
		if err != nil {
			return microerror.Mask(err)
		}

		ctrlClient, err = client.New(rest.CopyConfig(restConfig), client.Options{Scheme: schemeConfig, Mapper: mapper})
		if tenant.IsAPINotAvailable(err) {
			// Set status so we don't try to connect to the tenant cluster
			// again in this reconciliation loop.
			cc.Status.ClusterStatus.IsUnavailable = true

			r.logger.Debugf(ctx, "tenant API not available yet")
			r.logger.Debugf(ctx, "canceling resource")
			return nil

		} else if err != nil {
			return microerror.Mask(err)
		}
	}

	var k8sClient *kubernetes.Clientset
	{
		c := rest.CopyConfig(restConfig)

		k8sClient, err = kubernetes.NewForConfig(c)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	var helmClient helmclient.Interface
	{
		c := helmclient.Config{
			Fs:         r.fs,
			K8sClient:  k8sClient,
			Logger:     r.logger,
			RestClient: k8sClient.RESTClient(),
			RestConfig: restConfig,

			HTTPClientTimeout: r.httpClientTimeout,
		}

		helmClient, err = helmclient.New(c)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	cc.Clients = controllercontext.Clients{
		Ctrl: ctrlClient,
		Helm: helmClient,
	}

	return nil
}

// schemeBuilder is used to extend the known types of the client-go scheme.
type schemeBuilder []func(*runtime.Scheme) error
