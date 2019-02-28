package index

import (
	"context"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"

	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/app-operator/pkg/label"
	"github.com/giantswarm/app-operator/service/controller/appcatalog/v1/key"
)

func (r *StateGetter) GetDesiredState(ctx context.Context, obj interface{}) ([]*corev1.ConfigMap, error) {
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
		return nil, microerror.Mask(notFoundError)
	}

	defer response.Body.Close()
	content, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      configMapName(cr.Name),
			Namespace: r.indexNamespace,
			Labels: map[string]string{
				label.ManagedBy: r.projectName,
			},
		},
		Data: map[string]string{
			"index.yaml": string(content),
		},
	}

	return []*corev1.ConfigMap{configMap}, nil
}
