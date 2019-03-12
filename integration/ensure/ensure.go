// +build k8srequired

package ensure

import (
	"context"
	"fmt"
	"time"

	"github.com/giantswarm/backoff"
	"github.com/giantswarm/microerror"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/app-operator/integration/setup"
)

type CRTestCase int

const (
	Create CRTestCase = 0
	Update CRTestCase = 1
	Delete CRTestCase = 2
)

// waitForUpdatedChartCR will get an updated chart CR which has a resourceVersion greater than the one we have.
func WaitForUpdatedChartCR(ctx context.Context, cases CRTestCase, config *setup.Config, namespace, testAppReleaseName, resourceVersion string) error {
	operation := func() error {
		chart, err := config.Host.G8sClient().ApplicationV1alpha1().Charts(namespace).Get(testAppReleaseName, v1.GetOptions{})
		switch cases {
		case Create:
			if err != nil {
				return microerror.Mask(err)
			}
		case Update:
			if err != nil {
				return microerror.Mask(err)
			}
			if chart.ObjectMeta.ResourceVersion == resourceVersion {
				return microerror.Mask(testError)
			}
		case Delete:
			if errors.IsNotFound(err) {
				return nil
			} else {
				return microerror.Mask(err)
			}
		}
		return nil
	}
	notify := func(err error, t time.Duration) {
		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("failed to detect updated in chart CR: retrying in %s", t))
	}
	b := backoff.NewExponential(3*time.Minute, 10*time.Second)
	err := backoff.RetryNotify(operation, b, notify)
	if err != nil {
		return microerror.Mask(err)
	}
	return nil
}
