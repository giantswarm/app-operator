package cordonchart

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/giantswarm/apiextensions/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/apiextensions/pkg/clientset/versioned"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"k8s.io/apimachinery/pkg/types"

	"github.com/giantswarm/app-operator/pkg/annotation"
	"github.com/giantswarm/app-operator/service/controller/app/v1/key"
)

const (
	// Name is the identifier of the resource.
	Name = "cordonchartv1"
)

//Config represents the configuration used to create a new cordonchart resource.
type Config struct {
	// Dependencies.
	Logger micrologger.Logger

	// Settings.
	ChartNamespace string
}

type patchSpec struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

// Resource implements the cordonchart resource.
type Resource struct {
	// Dependencies.
	logger micrologger.Logger

	// Settings.
	chartNamespace string
}

// New creates a new configured cordonchart resource.
func New(config Config) (*Resource, error) {
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	if config.ChartNamespace == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.ChartNamespace must not be empty", config)
	}

	r := &Resource{
		logger:         config.Logger,
		chartNamespace: config.ChartNamespace,
	}

	return r, nil
}

func (r Resource) Name() string {
	return Name
}

func (r *Resource) addCordon(ctx context.Context, cr v1alpha1.App, client versioned.Interface) error {
	var err error
	var patchByte []byte
	{
		patch := []patchSpec{
			{
				Op:   "add",
				Path: "/metadata/annotations",
				Value: map[string]string{
					replacePrefix(annotation.CordonReason): key.CordonReason(cr),
					replacePrefix(annotation.CordonUntil):  key.CordonUntil(cr),
				},
			},
		}

		patchByte, err = json.Marshal(patch)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	_, err = client.ApplicationV1alpha1().Charts(r.chartNamespace).Patch(cr.GetName(), types.JSONPatchType, patchByte)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (r *Resource) deleteCordon(ctx context.Context, cr v1alpha1.App, client versioned.Interface) error {
	var err error
	var patchByte []byte

	{
		patch := []patchSpec{
			{
				Op:   "remove",
				Path: replaceToEscape(fmt.Sprintf("/metadata/annotations/%s", replacePrefix(annotation.CordonUntil))),
			},
			{
				Op:   "remove",
				Path: replaceToEscape(fmt.Sprintf("/metadata/annotations/%s", replacePrefix(annotation.CordonReason))),
			},
		}

		patchByte, err = json.Marshal(patch)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	_, err = client.ApplicationV1alpha1().Charts(r.chartNamespace).Patch(cr.GetName(), types.JSONPatchType, patchByte)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func replacePrefix(from string) string {
	return strings.Replace(from, "app-operator.giantswarm.io", "chart-operator.giantswarm.io", 1)
}

func replaceToEscape(from string) string {
	return strings.Replace(from, "/", "~1", -1)
}
