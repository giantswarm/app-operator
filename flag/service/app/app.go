package app

type App struct {
	DependencyWaitTimeoutMinutes string
	HelmControllerBackend        string
	Unique                       string
	WatchNamespace               string
	WorkloadClusterID            string
}
