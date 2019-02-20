package index

import (
	"context"
	"io/ioutil"
	"k8s.io/api/core/v1"
	"net/http"
	"net/url"
	"path"

	"github.com/giantswarm/microerror"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/app-operator/pkg/label"
	"github.com/giantswarm/app-operator/service/controller/appcatalog/v1/key"
)

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

	if response.StatusCode != 200 {
		return nil, microerror.Mask(indexNotFound)
	}

	defer response.Body.Close()
	content, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	configMap := &v1.ConfigMap{
		ObjectMeta: v12.ObjectMeta{
			Name:        cr.Name,
			Namespace:   cr.Namespace,
			Labels:      label.ProcessLabels(r.projectName, "", cr.ObjectMeta.Labels),
			Annotations: cr.ObjectMeta.Annotations,
		},
		Data: map[string]string{
			"index.yaml": string(content),
		},
	}

	return configMap, nil
}
