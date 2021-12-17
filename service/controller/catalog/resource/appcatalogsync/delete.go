package appcatalogsync

import (
	"context"

	"github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/app/v6/pkg/key"
	"github.com/giantswarm/microerror"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// EnsureDeleted ensures appcatalog CRs are deleted when catalog CRs are deleted.
func (r *Resource) EnsureDeleted(ctx context.Context, obj interface{}) error {
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

	var appCatalogCR v1alpha1.AppCatalog

	err = r.k8sClient.CtrlClient().Get(
		ctx,
		types.NamespacedName{Name: cr.Name},
		&appCatalogCR,
	)
	if apierrors.IsNotFound(err) {
		//no-op
		return nil
	}

	r.logger.Debugf(ctx, "deleting appCatalog %#q which had been created for compatibility", cr.GetName())

	err = r.k8sClient.CtrlClient().Delete(ctx, &appCatalogCR)
	if apierrors.IsNotFound(err) {
		// no-op
		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	r.logger.Debugf(ctx, "deleted appCatalog %#q which had been created for compatibility", cr.GetName())

	return nil
}
