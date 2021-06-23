package appcatalogsync

import (
	"context"

	"github.com/giantswarm/app/v5/pkg/key"
	"github.com/giantswarm/microerror"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EnsureDeleted ensures appcatalog CRs are deleted when catalog CRs are deleted.
func (r *Resource) EnsureDeleted(ctx context.Context, obj interface{}) error {
	if !r.uniqueApp {
		// Return early. Only unique instance manages appcatalogentry CRs.
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

	err = r.k8sClient.G8sClient().ApplicationV1alpha1().AppCatalogs().Delete(ctx, cr.GetName(), metav1.DeleteOptions{})
	if apierrors.IsNotFound(err) {
		// no-op
		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	r.logger.Debugf(ctx, "deleted appCatalog %#q which had been created for compatibility", cr.GetName())

	return nil
}
