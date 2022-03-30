package childapps

import (
	"context"
	"fmt"

	"github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/app/v6/pkg/key"
	"github.com/giantswarm/k8smetadata/pkg/label"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/v6/pkg/controller/context/finalizerskeptcontext"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	pkgstatus "github.com/giantswarm/app-operator/v5/pkg/status"
)

// EnsureDeleted checks if the app being deleted is a bundle with child apps.
// It sets the bundle app status and keeps its finalizer until all child apps
// are deleted so the deletion process is clearer to the user.
func (r *Resource) EnsureDeleted(ctx context.Context, obj interface{}) error {
	cr, err := key.ToApp(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	var apps []*v1alpha1.App
	{
		r.logger.Debugf(ctx, "finding child apps with label '%s=%s'", label.ManagedBy, cr.GetName())

		// Get all apps for cluster not being managed by cluster-apps-operator.
		selector, err := labels.Parse(fmt.Sprintf("%s=%s", label.ManagedBy, cr.GetName()))
		if err != nil {
			return microerror.Mask(err)
		}

		o := client.ListOptions{
			Namespace:     cr.GetNamespace(),
			LabelSelector: selector,
		}

		var appList v1alpha1.AppList

		err = r.ctrlClient.List(ctx, &appList, &o)
		if err != nil {
			return microerror.Mask(err)
		}

		for _, item := range appList.Items {
			apps = append(apps, item.DeepCopy())
		}
	}

	if len(apps) > 0 {
		message := fmt.Sprintf("waiting for %d child apps with label '%s=%s' to be deleted", len(apps), label.ManagedBy, cr.GetName())

		err = r.updateAppStatus(ctx, cr, message, pkgstatus.DeletingStatus)
		if err != nil {
			return microerror.Mask(err)
		}

		r.logger.Debugf(ctx, message)
		finalizerskeptcontext.SetKept(ctx)
		r.logger.Debugf(ctx, "keeping finalizers")

		return nil
	}

	r.logger.Debugf(ctx, "found %d child apps with label '%s=%s'", len(apps), label.ManagedBy, cr.GetName())

	return nil
}

func (r *Resource) updateAppStatus(ctx context.Context, cr v1alpha1.App, reason, status string) error {
	r.logger.Debugf(ctx, "setting status for app %#q in namespace %#q", cr.Name, cr.Namespace)

	var currentCR v1alpha1.App

	// Get app CR again to ensure the resource version is correct.
	err := r.ctrlClient.Get(
		ctx,
		types.NamespacedName{Name: cr.Name, Namespace: cr.Namespace},
		&currentCR,
	)
	if err != nil {
		return microerror.Mask(err)
	}

	currentCR.Status = v1alpha1.AppStatus{
		Release: v1alpha1.AppStatusRelease{
			Reason: reason,
			Status: status,
		},
	}

	err = r.ctrlClient.Status().Update(ctx, &currentCR)
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.Debugf(ctx, "status set for app %#q in namespace %#q", cr.Name, cr.Namespace)

	return nil
}
