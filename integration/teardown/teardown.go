// +build k8srequired

package teardown

import (
	"context"

	"github.com/giantswarm/e2e-harness/pkg/framework"
	"github.com/giantswarm/helmclient"
	"github.com/giantswarm/microerror"
	"k8s.io/helm/pkg/helm"
)

func Teardown(f *framework.Host, helmClient *helmclient.Client) error {
	// clean host cluster components
	err := framework.HelmCmd("delete --purge giantswarm-app-operator")
	if err != nil {
		return microerror.Mask(err)
	}

	// clean guest cluster components
	items := []string{"apiextensions-chart-e2e"}

	for _, item := range items {
		err := helmClient.DeleteRelease(context.TODO(), item, helm.DeletePurge(true))
		if err != nil {
			return microerror.Mask(err)
		}
	}
	return nil
}
