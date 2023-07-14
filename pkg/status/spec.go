package status

import (
	helmv2 "github.com/fluxcd/helm-controller/api/v2beta1"
	fluxmeta "github.com/fluxcd/pkg/apis/meta"
	"github.com/giantswarm/helmclient/v4/pkg/helmclient"
)

const (
	// AlreadyExistsStatus is set in the CR status when it failed to create
	// a manifest object because it exists already.
	AlreadyExistsStatus = "already-exists"

	// AppNotFoundStatus is set in the CR status when the app in question
	// cannot be found in the index.yaml of a catalog.
	AppNotFoundStatus = "app-not-found"

	// AppVersionNotFoundStatus is set in the CR status when the app in question
	// is found in the index.yaml of a catalog, but the given version is not.
	AppVersionNotFoundStatus = "app-version-not-found"

	// CatalogEmptyStatus is set in the CR status when the catalog's index.yaml
	// has no entries or is nil.
	CatalogEmptyStatus = "catalog-empty"

	// ChartPullFailedStatus is set in the CR status when it failed to pull
	// chart tarball for various reasons: network issues, tarball does not
	// exists, connection timeout etc.
	ChartPullFailedStatus = "chart-pull-failed"

	// ConfigmapMergeFailedStatus is set in the CR status when there is an failure during
	// merge configmaps.
	ConfigmapMergeFailedStatus = "configmap-merge-failed"

	// IndexNotFoundStatus is set in the CR status when the catalog's index.yaml
	// has no entries or is nil.
	IndexNotFoundStatus = "index-not-found"

	// InvalidManifestStatus is set in the CR status when it failed to create
	// manifest objects with helm resources.
	InvalidManifestStatus = "invalid-manifest"

	// PendingStatus is set in the CR status when the latest known status of
	// HelmRelease CR is known to be progressing without any further information.
	PendingStatus = "pending"

	// ReleaseNotInstalledStatus is set in the CR status when there is no Helm
	// Release to check.
	ReleaseNotInstalledStatus = "not-installed"

	// ResourceNotFoundStatus is set in the CR status when there is an failure during
	// finding dependents kubernete resources.
	ResourceNotFoundStatus = "resource-not-found"

	// SecretMergeFailedStatus is set in the CR status when there is an failure during
	// merge secrets.
	SecretMergeFailedStatus = "secret-merge-failed"

	// UnknownError is set in the CR status when there is an failure during
	// merge secrets.
	UnknownError = "unknown-error"

	// ValidationFailedStatus is set in the CR status when it failed to pass
	// OpenAPI validation on release manifest.
	ValidationFailedStatus = "validation-failed"

	// ValuesSchemaViolation is set in the CR status when Helm Chart has not passed
	// values.yaml validation against schema.
	ValuesSchemaViolation = "values-schema-violation"
)

var (
	FailedStatus = map[string]bool{
		ConfigmapMergeFailedStatus: true,
		SecretMergeFailedStatus:    true,
	}

	releasedReasonSuccessMapping = map[string]string{
		helmv2.InstallSucceededReason:   helmclient.StatusDeployed,
		helmv2.UpgradeSucceededReason:   helmclient.StatusDeployed,
		helmv2.RollbackSucceededReason:  helmclient.StatusDeployed,
		helmv2.UninstallSucceededReason: helmclient.StatusUninstalled,
	}

	releasedReasonFailMapping = map[string]string{
		helmv2.InstallFailedReason:   helmclient.StatusFailed,
		helmv2.UpgradeFailedReason:   helmclient.StatusFailed,
		helmv2.RollbackFailedReason:  helmclient.StatusFailed,
		helmv2.UninstallFailedReason: helmclient.StatusFailed,
	}

	readyReasonMapping = map[string]string{
		// The ArtifactFailedReason does not provide any granularity unfortunately.
		// It just informs of problem with the artifact, but does not say anything
		// about its nature, hence we can't infer from it wheather it is a pulling
		// problem, missing version, missing chart, or something else.
		// Fortunately, the "helmrelease" resource of App Operator supports the
		// feature of fallback repositories the same way the "chart" resource does it,
		// what means it checks for availability of Helm Chart in repository and hence
		// recognizes some of the possible problems. Hence we should not have many hits
		// for this reason here, if any. In case we get to this point, I believe we can
		// rely on the "chart-pull-failed" which according to its description, and
		// Chart Operator's code, has previously been used as a general indicator of
		// artifact-related problem, hence it may be continue to be used as such.
		helmv2.ArtifactFailedReason:       ChartPullFailedStatus,
		helmv2.InitFailedReason:           helmclient.StatusFailed,
		helmv2.GetLastReleaseFailedReason: helmclient.StatusFailed,

		// In theory, I believe we should never encounter the ReconciliationSucceededReason
		// reason, because it is set only when the Helm actions has been successfully executed,
		// what in turns means, there must be "Released" condition on the list, which we should
		// find and hence we should never get to the point of quering this map. But just in case
		// we map this reason as well here, and I think it makes sense to map it to "deployed"
		// status due to what it indicates.
		//
		// About the ReconciliationFailedReason, I believe it is very unlikely to be queried in
		// this map. If reconciliation failed it should be either due to one of the three reasons
		// above, or due to Helm action itself, which should be reported by the "Released" condition.
		// Just in case there is no such condition, and the reason for "Ready" condition is unknown,
		// we reflect that by the "unknown" status.
		helmv2.ReconciliationSucceededReason: helmclient.StatusDeployed,
		helmv2.ReconciliationFailedReason:    helmclient.StatusUnknown,
		fluxmeta.ProgressingReason:           "pending",
	}
)
