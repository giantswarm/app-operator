package configmap

import (
	"context"
	"fmt"

	"github.com/giantswarm/apiextensions/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/helmclient"
	"github.com/giantswarm/microerror"
	yaml "gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/app-operator/pkg/label"
	"github.com/giantswarm/app-operator/service/controller/app/v1/controllercontext"
	"github.com/giantswarm/app-operator/service/controller/app/v1/key"
	appcatalogkey "github.com/giantswarm/app-operator/service/controller/appcatalog/v1/key"
)

func (r *Resource) GetDesiredState(ctx context.Context, obj interface{}) (interface{}, error) {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	appConfigMapName := key.AppConfigMapName(cr)
	catalogConfigMapName := appcatalogkey.ConfigMapName(cc.AppCatalog)

	if appConfigMapName == "" && catalogConfigMapName == "" {
		// Return early as there is no config.
		return nil, nil
	}

	// We get the catalog level values if configured.
	catalogData, err := r.getConfigMapForCatalog(ctx, cc.AppCatalog)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	// We get the app level values if configured.
	appData, err := r.getConfigMapForApp(ctx, cr)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	// Values are merged and in case of intersecting values the app level
	// secrets are preferred.
	mergedData, err := helmclient.MergeValues(toByteSliceMap(catalogData), toByteSliceMap(appData))
	if err != nil {
		return nil, microerror.Mask(err)
	}

	valuesYAML, err := yaml.Marshal(mergedData)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	configMap := &corev1.ConfigMap{
		Data: map[string]string{
			"values": string(valuesYAML),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      key.ChartConfigMapName(cr),
			Namespace: key.Namespace(cr),
			Labels: map[string]string{
				label.ManagedBy: r.projectName,
			},
		},
	}

	return configMap, nil
}

func (r *Resource) getConfigMap(ctx context.Context, configMapName, configMapNamespace string) (map[string]string, error) {
	if configMapName == "" {
		// Return early.
		return map[string]string{}, nil
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("looking for configmap %#q in namespace %#q", configMapName, configMapNamespace))

	configMap, err := r.k8sClient.CoreV1().ConfigMaps(configMapNamespace).Get(configMapName, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return nil, microerror.Maskf(notFoundError, "configmap %#q in namespace %#q not found", configMapName, configMapNamespace)
	} else if err != nil {
		return nil, microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("found configmap %#q in namespace %#q", configMapName, configMapNamespace))

	return configMap.Data, nil
}

func (r *Resource) getConfigMapForApp(ctx context.Context, app v1alpha1.App) (map[string]string, error) {
	configMap, err := r.getConfigMap(ctx, key.AppConfigMapName(app), key.AppConfigMapNamespace(app))
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return configMap, nil
}

func (r *Resource) getConfigMapForCatalog(ctx context.Context, catalog v1alpha1.AppCatalog) (map[string]string, error) {
	configMap, err := r.getConfigMap(ctx, appcatalogkey.ConfigMapName(catalog), appcatalogkey.ConfigMapNamespace(catalog))
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return configMap, nil
}
