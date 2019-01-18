package chart

import (
	"context"
	"fmt"

	v1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/apiextensions/pkg/clientset/versioned"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/controller"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/giantswarm/app-operator/service/controller/app/v1/key"
)

const (
	// Name is the identifier of the resource.
	chartAPIVersion           = "application.giantswarm.io"
	chartKind                 = "Chart"
	chartVersionBundleVersion = "0.1.0"
	Name                      = "chartv1"
)

// Config represents the configuration used to create a new chart resource.
type Config struct {
	// Dependencies.
	G8sClient versioned.Interface
	K8sClient kubernetes.Interface
	Logger    micrologger.Logger
}

// Resource implements the chart resource.
type Resource struct {
	// Dependencies.
	g8sClient versioned.Interface
	k8sClient kubernetes.Interface
	logger    micrologger.Logger
}

func (r *Resource) GetCurrentState(ctx context.Context, obj interface{}) (interface{}, error) {
	fmt.Println("it's created!!!")
	return nil, nil
}

func (r *Resource) GetDesiredState(ctx context.Context, obj interface{}) (interface{}, error) {
	fmt.Println("it's desired state!!!")
	customResource, err := key.ToCustomResource(obj)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	catalogName := key.CatalogName(customResource)

	appCatalog, err := r.g8sClient.ApplicationV1alpha1().AppCatalogs("default").Get(catalogName, v1.GetOptions{})
	if err != nil {
		return nil, microerror.Mask(err)
	}

	chartURL := generateCatalogURL(appCatalog.Spec.CatalogStorage.URL, customResource.Spec.Name, customResource.Spec.Release)

	chartCR := &v1alpha1.Chart{
		TypeMeta: metav1.TypeMeta{
			Kind:       chartKind,
			APIVersion: chartAPIVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        customResource.Spec.Name,
			Labels:      customResource.GetObjectMeta().GetLabels(),
			Annotations: customResource.GetObjectMeta().GetAnnotations(),
		},
		Spec: v1alpha1.ChartSpec{
			Name:       customResource.GetObjectMeta().GetName(),
			Namespace:  customResource.Spec.Namespace,
			TarballURL: chartURL,
		},
	}

	if customResource.Spec.KubeConfig != (v1alpha1.AppSpecKubeConfig{}) {
		chartCR.Spec.KubeConfig.Secret.Name = customResource.Spec.KubeConfig.Secret.Name
		chartCR.Spec.KubeConfig.Secret.Namespace = customResource.Spec.KubeConfig.Secret.Namespace
	}

	if customResource.Spec.Config != (v1alpha1.AppSpecConfig{}) {
		chartCR.Spec.Config.Secret.Name = customResource.Spec.Config.Secret.Name
		chartCR.Spec.Config.Secret.Namespace = customResource.Spec.Config.Secret.Namespace

		chartCR.Spec.Config.ConfigMap.Name = customResource.Spec.Config.ConfigMap.Name
		chartCR.Spec.Config.ConfigMap.Namespace = customResource.Spec.Config.ConfigMap.Namespace
	}
	return chartCR, nil
}

func generateCatalogURL(baseURL string, appName string, release string) string {
	return fmt.Sprintf("%s-%s-%s.tgz", baseURL, appName, release)
}

func (r *Resource) NewUpdatePatch(ctx context.Context, obj, currentState, desiredState interface{}) (*controller.Patch, error) {
	patch := controller.NewPatch()
	patch.SetCreateChange(desiredState)

	return patch, nil
}

func (r *Resource) NewDeletePatch(ctx context.Context, obj, currentState, desiredState interface{}) (*controller.Patch, error) {
	fmt.Println("it's patched!!!")
	return nil, nil
}

func (r *Resource) ApplyCreateChange(ctx context.Context, obj, createChange interface{}) error {
	return nil
}

func (r *Resource) ApplyDeleteChange(ctx context.Context, obj, deleteChange interface{}) error {
	return nil
}

func (r *Resource) ApplyUpdateChange(ctx context.Context, obj, updateChange interface{}) error {
	return nil
}

// New creates a new configured chart resource.
func New(config Config) (*Resource, error) {
	// Dependencies.
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

func (r *Resource) Name() string {
	return Name
}
