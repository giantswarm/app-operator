package app

type App struct {
	DependencyWaitTimeoutMinutes       string
	HelmControllerBackend              string
	HelmControllerBackendAutoMigration string
	Unique                             string
	WatchNamespace                     string
	WorkloadClusterID                  string
}
