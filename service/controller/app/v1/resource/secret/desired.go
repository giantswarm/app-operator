package secret

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
	"github.com/giantswarm/app-operator/service/controller/app/v1/values"
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

	appSecretName := key.AppSecretName(cr)
	catalogSecretName := appcatalogkey.SecretName(cc.AppCatalog)
	userSecretName := key.UserSecretName(cr)

	if appSecretName == "" && catalogSecretName == "" && userSecretName == "" {
		// Return early as there is no secret.
		return nil, nil
	}

	// We get the catalog level secrets if configured.
	catalogData, err := r.getSecretDataForCatalog(ctx, cc.AppCatalog)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	// We get the app level secrets if configured.
	appData, err := r.getSecretDataForApp(ctx, cr)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	// Secrets are merged and in case of intersecting values the app level
	// secrets are preferred.
	mergedData, err := values.MergeSecretData(catalogData, appData)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	// We get the user level values if configured and merge them.
	if userSecretName != "" {
		userData, err := r.getUserSecretDataForApp(ctx, cr)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		// Secrets are merged again and in case of intersecting values the user
		// level secrets are preferred.
		mergedData, err = values.MergeSecretData(mergedData, userData)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	secret := &corev1.Secret{
		Data: mergedData,
		ObjectMeta: metav1.ObjectMeta{
			Name:      key.ChartSecretName(cr),
			Namespace: r.chartNamespace,
			Labels: map[string]string{
				label.ManagedBy: r.projectName,
			},
		},
	}

	return secret, nil
}

func (r *Resource) getSecret(ctx context.Context, secretName, secretNamespace string) (map[string][]byte, error) {
	if secretName == "" {
		// Return early as no secret has been specified.
		return nil, nil
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("looking for secret %#q in namespace %#q", secretName, secretNamespace))

	secret, err := r.k8sClient.CoreV1().Secrets(secretNamespace).Get(secretName, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return nil, microerror.Maskf(notFoundError, "secret %#q in namespace %#q not found", secretName, secretNamespace)
	} else if err != nil {
		return nil, microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("found secret %#q in namespace %#q", secretName, secretNamespace))

	return secret.Data, nil
}

func (r *Resource) getSecretDataForApp(ctx context.Context, app v1alpha1.App) (map[string][]byte, error) {
	secret, err := r.getSecret(ctx, key.AppSecretName(app), key.AppSecretNamespace(app))
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return secret, nil
}

func (r *Resource) getSecretDataForCatalog(ctx context.Context, catalog v1alpha1.AppCatalog) (map[string][]byte, error) {
	secret, err := r.getSecret(ctx, appcatalogkey.SecretName(catalog), appcatalogkey.SecretNamespace(catalog))
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return secret, nil
}

func (r *Resource) getUserSecretDataForApp(ctx context.Context, app v1alpha1.App) (map[string][]byte, error) {
	secret, err := r.getSecret(ctx, key.UserSecretName(app), key.UserSecretNamespace(app))
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return secret, nil
}
