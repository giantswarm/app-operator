package values

import (
	"context"
	"fmt"

	"github.com/giantswarm/apiextensions/v3/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/app/v3/pkg/key"
	"github.com/giantswarm/helmclient/v3/pkg/helmclient"
	"github.com/giantswarm/microerror"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MergeConfigMapData merges the data from the catalog, app and user configmaps
// and returns a single set of values.
func (v *Values) MergeConfigMapData(ctx context.Context, app v1alpha1.App, appCatalog v1alpha1.AppCatalog) (map[string]string, error) {
	appConfigMapName := key.AppConfigMapName(app)
	catalogConfigMapName := key.AppCatalogConfigMapName(appCatalog)
	userConfigMapName := key.UserConfigMapName(app)

	if appConfigMapName == "" && catalogConfigMapName == "" && userConfigMapName == "" {
		// Return early as there is no config.
		return nil, nil
	}

	// We get the catalog level values if configured.
	catalogData, err := v.getConfigMapForCatalog(ctx, appCatalog)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	// We get the app level values if configured.
	appData, err := v.getConfigMapForApp(ctx, app)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	// Config is merged and in case of intersecting values the app level
	// config is preferred.
	mergedData, err := mergeConfigMapData(catalogData, appData)
	if helmclient.IsParsingDestFailedError(err) {
		return nil, microerror.Maskf(parsingError, "failed to parse catalog configmap, logs from merging: %s", err.Error())
	} else if helmclient.IsParsingSrcFailedError(err) {
		return nil, microerror.Maskf(parsingError, "failed to parse app configmap, logs from merging: %s", err.Error())
	} else if err != nil {
		return nil, microerror.Mask(err)
	}

	// We get the user level values if configured and merge them.
	if key.UserConfigMapName(app) != "" {
		userData, err := v.getUserConfigMapForApp(ctx, app)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		// Config is merged again and in case of intersecting values the user
		// level config is preferred.
		mergedData, err = mergeConfigMapData(mergedData, userData)
		if helmclient.IsParsingDestFailedError(err) {
			return nil, microerror.Maskf(parsingError, "failed to parse previous merged configmap, logs from merging: %s", err.Error())
		} else if helmclient.IsParsingSrcFailedError(err) {
			return nil, microerror.Maskf(parsingError, "failed to parse user configmap, logs from merging: %s", err.Error())
		} else if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	return mergedData, nil
}

func (v *Values) getConfigMap(ctx context.Context, configMapName, configMapNamespace string) (map[string]string, error) {
	if configMapName == "" {
		// Return early as no configmap has been specified.
		return nil, nil
	}

	v.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("looking for configmap %#q in namespace %#q", configMapName, configMapNamespace))

	configMap, err := v.k8sClient.CoreV1().ConfigMaps(configMapNamespace).Get(ctx, configMapName, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return nil, microerror.Maskf(notFoundError, "configmap %#q in namespace %#q not found", configMapName, configMapNamespace)
	} else if err != nil {
		return nil, microerror.Mask(err)
	}

	v.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("found configmap %#q in namespace %#q", configMapName, configMapNamespace))

	return configMap.Data, nil
}

func (v *Values) getConfigMapForApp(ctx context.Context, app v1alpha1.App) (map[string]string, error) {
	configMap, err := v.getConfigMap(ctx, key.AppConfigMapName(app), key.AppConfigMapNamespace(app))
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return configMap, nil
}

func (v *Values) getConfigMapForCatalog(ctx context.Context, catalog v1alpha1.AppCatalog) (map[string]string, error) {
	configMap, err := v.getConfigMap(ctx, key.AppCatalogConfigMapName(catalog), key.AppCatalogConfigMapNamespace(catalog))
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return configMap, nil
}

func (v *Values) getUserConfigMapForApp(ctx context.Context, app v1alpha1.App) (map[string]string, error) {
	configMap, err := v.getConfigMap(ctx, key.UserConfigMapName(app), key.UserConfigMapNamespace(app))
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return configMap, nil
}

// mergeConfigMapData merges configmap data into a single block of YAML that
// is stored in the configmap associated with the relevant chart CR.
func mergeConfigMapData(destMap, srcMap map[string]string) (map[string]string, error) {
	result, err := mergeData(toByteSliceMap(destMap), toByteSliceMap(srcMap))
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return toStringMap(result), nil
}
