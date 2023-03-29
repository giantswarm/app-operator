package status

const (
	// appNotFoundStatus is set in the CR status when it app in question
	// cannot be found in the index.yaml of a catalog.
	AppNotFoundStatus = "app-not-found"

	// appVersionNotFoundStatus is set in the CR status when it app in question
	// is found in the index.yaml of a catalog, but the version is not.
	AppVersionNotFoundStatus = "app-version-not-found"

	// catalogEmptyStatus is set in the CR status when the catalog's index.yaml
	// has no entries.
	CatalogEmptyStatus = "catalog-empty"

	// ConfigmapMergeFailedStatus is set in the CR status when there is an failure during
	// merge configmaps.
	ConfigmapMergeFailedStatus = "configmap-merge-failed"

	// ResourceNotFoundStatus is set in the CR status when there is an failure during
	// finding dependents kubernete resources.
	ResourceNotFoundStatus = "resource-not-found"

	// SecretMergeFailedStatus is set in the CR status when there is an failure during
	// merge secrets.
	SecretMergeFailedStatus = "secret-merge-failed"
)

var (
	FailedStatus = map[string]bool{
		ConfigmapMergeFailedStatus: true,
		SecretMergeFailedStatus:    true,
	}
)
