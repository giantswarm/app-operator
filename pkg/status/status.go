package status

const (
	// ConfigmapMergeFailedStatus is set in the CR status when there is an failure during
	// merge configmaps.
	ConfigmapMergeFailedStatus = "configmap-merge-failed"

	// DeletingStatus is set in the CR status while child apps are being deleted.
	DeletingStatus = "deleting"

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
