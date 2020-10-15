package appcatalogentry

import (
	"context"
	"fmt"

	"github.com/giantswarm/apiextensions/v2/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/microerror"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/app-operator/v2/service/controller/appcatalog/key"
)

// EnsureDeleted ensures appcatalogentry CRs are deleted for this appcatalog CR.
func (r *Resource) EnsureDeleted(ctx context.Context, obj interface{}) error {
	if !r.uniqueApp {
		// Return early. Only unique instance manages appcatalogentry CRs.
		return nil
	}

	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	if key.CatalogVisibility(cr) != publicVisibilityType {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("not deleting CRs for catalog %#q with visibility %#q", cr.Name, key.CatalogVisibility(cr)))
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
		return nil
	}

	entryCRs, err := r.getCurrentEntryCRs(ctx, cr)
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("deleting %d appcatalogentry CR %#q in namespace %#q", len(entryCRs), cr.Name, cr.Namespace))

	for _, entryCR := range entryCRs {
		err = r.deleteAppCatalogEntry(ctx, entryCR)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("deleted %d appcatalogentry CR %#q in namespace %#q", len(entryCRs), cr.Name, cr.Namespace))

	return nil
}

func (r *Resource) deleteAppCatalogEntry(ctx context.Context, entryCR *v1alpha1.AppCatalogEntry) error {
	err := r.k8sClient.G8sClient().ApplicationV1alpha1().AppCatalogEntries(entryCR.Namespace).Delete(ctx, entryCR.Name, metav1.DeleteOptions{})
	if apierrors.IsNotFound(err) {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("already deleted appcatalogentry CR %#q in namespace %#q", entryCR.Name, entryCR.Namespace))
		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
