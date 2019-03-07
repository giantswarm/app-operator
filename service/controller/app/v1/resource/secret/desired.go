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

	if appSecretName == "" && catalogSecretName == "" {
		// Return early as there is no secret.
		return nil, nil
	}

	if appSecretName != "" && catalogSecretName != "" {
		return nil, microerror.Maskf(executionFailedError, "merging app and catalog secrets is not yet supported")
	}

	data, err := r.getSecretData(ctx, cr)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	secret := &corev1.Secret{
		Data: data,
		ObjectMeta: metav1.ObjectMeta{
			Name:      key.ChartSecretName(cr),
			Namespace: key.Namespace(cr),
			Labels: map[string]string{
				label.ManagedBy: r.projectName,
			},
		},
	}

	return secret, nil
}

func (r *Resource) getSecret(ctx context.Context, secretName, secretNamespace string) (*corev1.Secret, error) {
	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("looking for secret %#q in namespace %#q", secretName, secretNamespace))

	secret, err := r.k8sClient.CoreV1().Secrets(secretNamespace).Get(secretName, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return nil, microerror.Maskf(notFoundError, "secret %#q in namespace %#q not found", secretName, secretNamespace)
	} else if err != nil {
		return nil, microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("found secret %#q in namespace %#q", secretName, secretNamespace))

	return secret, nil
}

func (r *Resource) getSecretData(ctx context.Context, cr v1alpha1.App) (map[string][]byte, error) {
	data, err := r.getSecretForApp(ctx, cr)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	if len(data) > 0 {
		return data, nil
	}

	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	data, err = r.getSecretForCatalog(ctx, cc.AppCatalog)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return data, nil
}

func (r *Resource) getSecretForApp(ctx context.Context, app v1alpha1.App) (map[string][]byte, error) {
	secret, err := r.getSecret(ctx, key.AppSecretName(app), key.AppSecretNamespace(app))
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return secret.Data, nil
}

func (r *Resource) getSecretForCatalog(ctx context.Context, catalog v1alpha1.AppCatalog) (map[string][]byte, error) {
	secret, err := r.getSecret(ctx, appcatalogkey.SecretName(catalog), appcatalogkey.SecretNamespace(catalog))
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return secret.Data, nil
}
