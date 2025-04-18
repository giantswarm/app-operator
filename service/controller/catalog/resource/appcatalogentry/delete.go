package appcatalogentry

import (
	"context"

	"github.com/giantswarm/app/v8/pkg/key"
	"github.com/giantswarm/microerror"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

// EnsureDeleted ensures appcatalogentry CRs are deleted for this catalog CR.
func (r *Resource) EnsureDeleted(ctx context.Context, obj interface{}) error {
	if !r.uniqueApp {
		// Return early. Only unique instance manages appcatalogentry CRs.
		return nil
	}

	cr, err := key.ToCatalog(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	entryCRs, err := r.getCurrentEntryCRs(ctx, cr)
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.Debugf(ctx, "deleting %d appcatalogentry CR %#q in namespace %#q", len(entryCRs), cr.Name, cr.Namespace)

	for _, entryCR := range entryCRs {
		err := r.k8sClient.CtrlClient().Delete(ctx, entryCR)
		if apierrors.IsNotFound(err) {
			r.logger.Debugf(ctx, "already deleted appcatalogentry CR %#q in namespace %#q", entryCR.Name, entryCR.Namespace)
			continue
		} else if err != nil {
			return microerror.Mask(err)
		}
	}

	r.logger.Debugf(ctx, "deleted %d appcatalogentry CR %#q in namespace %#q", len(entryCRs), cr.Name, cr.Namespace)

	return nil
}
