package release

import (
	"context"
	"fmt"
	"time"

	"github.com/giantswarm/backoff"
	"github.com/giantswarm/helmclient"
	"github.com/giantswarm/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Config struct {
	HelmClient helmclient.Interface
	K8sClient  k8sclient.Interface
	Logger     micrologger.Logger
}

type Release struct {
	helmClient helmclient.Interface
	k8sClient  k8sclient.Interface
	logger     micrologger.Logger
}

func New(config Config) (*Release, error) {
	if config.HelmClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.HelmClient must not be empty", config)
	}
	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	r := &Release{
		helmClient: config.HelmClient,
		k8sClient:  config.K8sClient,
		logger:     config.Logger,
	}

	return r, nil
}

func (r *Release) PodExists(ctx context.Context, namespace, labelSelector string) error {
	o := func() error {
		pods, err := r.k8sClient.K8sClient().CoreV1().Pods(namespace).List(metav1.ListOptions{LabelSelector: labelSelector})
		if err != nil {
			return microerror.Mask(err)
		}
		if len(pods.Items) != 1 {
			return microerror.Maskf(waitError, "expected 1 pod but got %d", len(pods.Items))
		}

		pod := pods.Items[0]
		if pod.Status.Phase != corev1.PodRunning {
			return microerror.Maskf(waitError, "expected Pod phase %#q but got %#q", corev1.PodRunning, pod.Status.Phase)
		}

		return nil
	}

	n := func(err error, t time.Duration) {
		r.logger.Log("level", "debug", "message", fmt.Sprintf("failed to get pod with selector '%s': retrying in %s", labelSelector, t), "stack", fmt.Sprintf("%v", err))
	}

	b := backoff.NewExponential(2*time.Minute, 60*time.Second)
	err := backoff.RetryNotify(o, b, n)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (r *Release) WaitForChartVersion(ctx context.Context, namespace, release, version string) error {
	o := func() error {
		rh, err := r.helmClient.GetReleaseContent(ctx, namespace, release)
		if err != nil {
			return microerror.Mask(err)
		}
		if rh.Version != version {
			return microerror.Maskf(releaseVersionNotMatchingError, "waiting for '%s', current '%s'", version, rh.Version)
		}
		return nil
	}

	n := func(err error, t time.Duration) {
		r.logger.Log("level", "debug", "message", fmt.Sprintf("failed to get release version '%s': retrying in %s", version, t), "stack", fmt.Sprintf("%v", err))
	}

	b := backoff.NewExponential(2*time.Minute, 60*time.Second)
	err := backoff.RetryNotify(o, b, n)
	if err != nil {
		return microerror.Mask(err)
	}
	return nil
}

func (r *Release) WaitForDeletedApp(ctx context.Context, namespace, appName string) error {
	o := func() error {
		_, err := r.k8sClient.G8sClient().ApplicationV1alpha1().Apps(namespace).Get(appName, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			// Fall through.
			return nil
		} else if err != nil {
			return microerror.Mask(err)
		}

		return nil
	}

	n := func(err error, t time.Duration) {
		r.logger.Log("level", "debug", "message", fmt.Sprintf("failed to get deleted app '%s': retrying in %s", appName, t), "stack", fmt.Sprintf("%v", err))
	}

	b := backoff.NewExponential(2*time.Minute, 60*time.Second)
	err := backoff.RetryNotify(o, b, n)
	if err != nil {
		return microerror.Mask(err)
	}
	return nil
}

func (r *Release) WaitForDeletedChart(ctx context.Context, namespace, chartName string) error {
	o := func() error {
		_, err := r.k8sClient.G8sClient().ApplicationV1alpha1().Charts(namespace).Get(chartName, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			// Fall through.
			return nil
		} else if err != nil {
			return microerror.Mask(err)
		}

		return nil
	}

	n := func(err error, t time.Duration) {
		r.logger.Log("level", "debug", "message", fmt.Sprintf("failed to get deleted chart '%s': retrying in %s", chartName, t), "stack", fmt.Sprintf("%v", err))
	}

	b := backoff.NewExponential(2*time.Minute, 60*time.Second)
	err := backoff.RetryNotify(o, b, n)
	if err != nil {
		return microerror.Mask(err)
	}
	return nil
}

func (r *Release) WaitForStatus(ctx context.Context, namespace, release, status string) error {
	o := func() error {
		rc, err := r.helmClient.GetReleaseContent(ctx, namespace, release)
		if helmclient.IsReleaseNotFound(err) && status == helmclient.StatusUninstalled {
			// Error is expected because we purge releases when deleting.
			return nil
		} else if err != nil {
			return microerror.Mask(err)
		}
		if rc.Status != status {
			return microerror.Maskf(releaseStatusNotMatchingError, "waiting for '%s', current '%s'", status, rc.Status)
		}
		return nil
	}

	n := func(err error, t time.Duration) {
		r.logger.Log("level", "debug", "message", fmt.Sprintf("failed to get release status '%s': retrying in %s", status, t), "stack", fmt.Sprintf("%v", err))
	}

	b := backoff.NewExponential(2*time.Minute, 60*time.Second)
	err := backoff.RetryNotify(o, b, n)
	if err != nil {
		return microerror.Mask(err)
	}
	return nil
}
