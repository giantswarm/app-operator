package appcatalogsync

import (
	"context"

	"github.com/giantswarm/apiextensions/v3/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/app/v5/pkg/key"
	"github.com/giantswarm/microerror"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EnsureCreated ensures appcatalog CRs are created for compatibility with catalog CRs
func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	if !r.uniqueApp {
		// Return early. Only unique instance manages appcatalog CRs.
		return nil
	}

	cr, err := key.ToCatalog(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	if namespace := cr.GetNamespace(); namespace == metav1.NamespaceDefault || namespace == "giantswarm" {
		// Return early. No need to reconcile catalog CRs in default namespace or giantswarm namespace.
		return nil
	}

	newAppCatalogCR := &v1alpha1.AppCatalog{
		ObjectMeta: metav1.ObjectMeta{
			Name:        cr.GetName(),
			Annotations: cr.GetAnnotations(),
			Labels:      cr.GetLabels(),
		},
		Spec: v1alpha1.AppCatalogSpec{
			Description: cr.Spec.Description,
			Title:       key.CatalogTitle(cr),
			Storage: v1alpha1.AppCatalogSpecStorage{
				Type: "helm",
				URL:  key.CatalogStorageURL(cr),
			},
		},
	}

	appCatalogCR, err := r.k8sClient.G8sClient().ApplicationV1alpha1().AppCatalogs().Get(ctx, cr.GetName(), metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		r.logger.Debugf(ctx, "creating appCatalog %#q for compatibility", cr.GetName())

		_, err = r.k8sClient.G8sClient().ApplicationV1alpha1().AppCatalogs().Create(ctx, appCatalogCR, metav1.CreateOptions{})
		if err != nil {
			return microerror.Mask(err)
		}

		r.logger.Debugf(ctx, "created appCatalog %#q for compatibility", cr.GetName())

		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	if !equals(appCatalogCR, newAppCatalogCR) {
		r.logger.Debugf(ctx, "updating appCatalog %#q for compatibility", cr.GetName())

		newAppCatalogCR.ResourceVersion = appCatalogCR.ResourceVersion
		_, err = r.k8sClient.G8sClient().ApplicationV1alpha1().AppCatalogs().Update(ctx, newAppCatalogCR, metav1.UpdateOptions{})
		if err != nil {
			return microerror.Mask(err)
		}

		r.logger.Debugf(ctx, "updated appCatalog %#q for compatibility", cr.GetName())
	}

	return nil
}
