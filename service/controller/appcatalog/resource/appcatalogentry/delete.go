package appcatalogentry

import (
	"context"
	"fmt"

	"github.com/giantswarm/app/v4/pkg/key"
	"github.com/giantswarm/microerror"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EnsureDeleted ensures appcatalogentry CRs are deleted for this appcatalog CR.
func (r *Resource) EnsureDeleted(ctx context.Context, obj interface{}) error {
	if !r.uniqueApp {
		// Return early. Only unique instance manages appcatalogentry CRs.
		return nil
	}

	cr, err := key.ToAppCatalog(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	entryCRs, err := r.getCurrentEntryCRs(ctx, cr)
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("deleting %d appcatalogentry CR %#q in namespace %#q", len(entryCRs), cr.Name, cr.Namespace))

	for _, entryCR := range entryCRs {
		err := r.k8sClient.G8sClient().ApplicationV1alpha1().AppCatalogEntries(entryCR.Namespace).Delete(ctx, entryCR.Name, metav1.DeleteOptions{})
		if apierrors.IsNotFound(err) {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("already deleted appcatalogentry CR %#q in namespace %#q", entryCR.Name, entryCR.Namespace))
			continue
		} else if err != nil {
			return microerror.Mask(err)
		}
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("deleted %d appcatalogentry CR %#q in namespace %#q", len(entryCRs), cr.Name, cr.Namespace))

	return nil
}
