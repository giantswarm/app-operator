package index

import (
	"context"
	"github.com/giantswarm/app-operator/pkg/label"
	"github.com/giantswarm/app-operator/service/controller/appcatalog/v1/key"
	"io/ioutil"
	"k8s.io/api/core/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
	"net/url"
	"path"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/controller"
	"k8s.io/client-go/kubernetes"
)

const (
	// Name is the identifier of the resource.
	Name = "indexv1"
)

// Config represents the configuration used to create a new index resource.
type Config struct {
	// Dependencies.
	K8sClient   kubernetes.Interface
	Logger      micrologger.Logger
	ProjectName string
}

// Resource implements the index resource.
type Resource struct {
	// Dependencies.
	k8sClient kubernetes.Interface
	logger    micrologger.Logger

	// Settings.
	projectName string
}

func (r *Resource) GetCurrentState(ctx context.Context, obj interface{}) (interface{}, error) {
	return nil, nil
}

func (r *Resource) GetDesiredState(ctx context.Context, obj interface{}) (interface{}, error) {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	storageURL, err := url.Parse(key.AppCatalogStorageURL(cr))
	if err != nil {
		return nil, microerror.Mask(err)
	}
	storageURL.Path = path.Join(storageURL.Path, "index.yaml")
	response, err := http.Get(storageURL.String())
	if err != nil {
		return nil, microerror.Mask(err)
	}

	defer response.Body.Close()
	content, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	configMap := &v1.ConfigMap{
		ObjectMeta: v12.ObjectMeta{
			Name:        cr.Name,
			Labels:      label.ProcessLabels(r.projectName, "", cr.ObjectMeta.Labels),
			Annotations: cr.ObjectMeta.Annotations,
		},
		Data: map[string]string{
			"index.yaml": string(content),
		},
	}

	return configMap, nil
}

func (r *Resource) NewUpdatePatch(ctx context.Context, obj, currentState, desiredState interface{}) (*controller.Patch, error) {
	return nil, nil
}

func (r *Resource) NewDeletePatch(ctx context.Context, obj, currentState, desiredState interface{}) (*controller.Patch, error) {
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

// New creates a new configured index resource.
func New(config Config) (*Resource, error) {
	// Dependencies.
	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}
	if config.ProjectName == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.ProjectName must not be empty", config)
	}

	r := &Resource{
		// Dependencies.
		k8sClient:   config.K8sClient,
		logger:      config.Logger,
		projectName: config.ProjectName,
	}

	return r, nil
}

func (r *Resource) Name() string {
	return Name
}
