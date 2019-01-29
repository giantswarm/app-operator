package status

import (
	"context"
	"fmt"

	"github.com/giantswarm/microerror"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/app-operator/service/controller/app/v1/key"
)

func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	name := key.AppName(cr)

	g8sClient, err := r.kubeConfig.NewG8sClientForApp(ctx, cr)
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("getting status for chart %#q", name))
	chart, err := g8sClient.ApplicationV1alpha1().Charts(r.watchNamespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return microerror.Maskf(notFoundError, "chart %#q in namespace %#q", name, r.watchNamespace)
	}

	if chart.Status.Status != "" && key.AppStatus(cr) != chart.Status.Status {
		r.logger.LogCtx(ctx, "level", "debug", "message",
			fmt.Sprintf("setting app %#q status as %#q", name, chart.Status.Status))

		customResourceCopy := cr.DeepCopy()
		customResourceCopy.Status.LastDeployed = *chart.Status.LastDeployed.DeepCopy()
		customResourceCopy.Status.Status = chart.Status.Status
		customResourceCopy.Status.Version = chart.Status.Version
		customResourceCopy.Status.AppVersion = chart.Status.AppVersion

		_, err = r.g8sClient.ApplicationV1alpha1().Apps(cr.Namespace).UpdateStatus(customResourceCopy)
		if err != nil {
			return microerror.Mask(err)
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("status set for chart %#q", name))
	} else {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("status for chart %#q already set to %#q", name, chart.Status.Status))
	}

	return nil
}
