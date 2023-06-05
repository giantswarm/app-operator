package status

const (
	// AppNotFoundStatus is set in the CR status when the app in question
	// cannot be found in the index.yaml of a catalog.
	AppNotFoundStatus = "app-not-found"

	// AppVersionNotFoundStatus is set in the CR status when the app in question
	// is found in the index.yaml of a catalog, but the given version is not.
	AppVersionNotFoundStatus = "app-version-not-found"

	// CatalogEmptyStatus is set in the CR status when the catalog's index.yaml
	// has no entries or is nil.
	CatalogEmptyStatus = "catalog-empty"

	// ConfigmapMergeFailedStatus is set in the CR status when there is an failure during
	// merge configmaps.
	ConfigmapMergeFailedStatus = "configmap-merge-failed"

	// IndexNotFoundStatus is set in the CR status when the catalog's index.yaml
	// has no entries or is nil.
	IndexNotFoundStatus = "index-not-found"

	// ResourceNotFoundStatus is set in the CR status when there is an failure during
	// finding dependents kubernete resources.
	ResourceNotFoundStatus = "resource-not-found"

	// SecretMergeFailedStatus is set in the CR status when there is an failure during
	// merge secrets.
	SecretMergeFailedStatus = "secret-merge-failed"

	// UnknownError is set in the CR status when there is an failure during
	// merge secrets.
	UnknownError = "unknown-error"
)

var (
	FailedStatus = map[string]bool{
		ConfigmapMergeFailedStatus: true,
		SecretMergeFailedStatus:    true,
	}
)
