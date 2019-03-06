package configmap

import (
	"context"
	"fmt"

	"github.com/giantswarm/apiextensions/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/microerror"
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

	if appConfigMapName != "" && catalogConfigMapName != "" {
		return nil, microerror.Maskf(executionFailedError, "merging app and catalog configmaps is not yet supported")
	}

	data, err := r.getConfigMapData(ctx, cr)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	configMap := &corev1.ConfigMap{
		Data: data,
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

func (r *Resource) getConfigMap(ctx context.Context, configMapName, configMapNamespace string) (*corev1.ConfigMap, error) {
	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("looking for configmap %#q in namespace %#q", configMapName, configMapNamespace))

	configMap, err := r.k8sClient.CoreV1().ConfigMaps(configMapNamespace).Get(configMapName, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return nil, microerror.Maskf(notFoundError, "config map %#q in namespace %#q not found", configMapName, configMapNamespace)
	} else if err != nil {
		return nil, microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("found configmap %#q in namespace %#q", configMapName, configMapNamespace))

	return configMap, nil
}

func (r *Resource) getConfigMapData(ctx context.Context, cr v1alpha1.App) (map[string]string, error) {
	data := make(map[string]string)

	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	appConfigMapName := key.AppConfigMapName(cr)
	catalogConfigMapName := appcatalogkey.ConfigMapName(cc.AppCatalog)

	if appConfigMapName != "" && catalogConfigMapName == "" {
		appConfigMap, err := r.getConfigMap(ctx, appConfigMapName, key.AppConfigMapNamespace(cr))
		if err != nil {
			return nil, microerror.Mask(err)
		}

		data = appConfigMap.Data
	}

	if appConfigMapName == "" && catalogConfigMapName != "" {
		catalogConfigMap, err := r.getConfigMap(ctx, catalogConfigMapName, appcatalogkey.ConfigMapNamespace(cc.AppCatalog))
		if err != nil {
			return nil, microerror.Mask(err)
		}

		data = catalogConfigMap.Data
	}

	return data, nil
}
