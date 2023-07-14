package helmreleasestatus

import (
	"context"
	"time"

	"github.com/giantswarm/backoff"
	"github.com/giantswarm/errors/tenant"
	"github.com/giantswarm/microerror"
	"k8s.io/client-go/dynamic"
)

// waitForAvailableConnection ensures we can connect to the target cluster if it
// is remote. Sometimes the connection will be unavailable so we list all HelmRelease
// CRs to confirm the connection is active.
func (c *HelmReleaseStatusWatcher) waitForAvailableConnection(ctx context.Context, dynClient dynamic.Interface) error {
	var err error

	listOption, err := c.getListOptions(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	o := func() error {
		// List all HelmRelease CRs in the target cluster to confirm the connection
		// is active and the chart CRD is installed.
		_, err = dynClient.Resource(helmReleaseResource).Namespace(c.podNamespace).List(ctx, listOption)
		if tenant.IsAPINotAvailable(err) {
			c.logger.Debugf(ctx, "workload cluster is not available")
			return microerror.Mask(err)
		} else if IsResourceNotFound(err) {
			c.logger.Debugf(ctx, "HelmRelease CRD is not installed")
			return microerror.Mask(err)
		} else if err != nil {
			return microerror.Mask(err)
		}

		return nil
	}

	n := func(err error, t time.Duration) {
		c.logger.Debugf(ctx, "failed to get available connection: %#v retrying in %s", err, t)
	}

	// maxWait is 0 since cluster creation may fail.
	b := backoff.NewExponential(0, 30*time.Second)
	err = backoff.RetryNotify(o, b, n)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
