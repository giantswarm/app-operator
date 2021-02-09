package annotation

const (
	// AppNamespace annotation is used by the chart status watcher to find the
	// app CR for chart CRs which are always in the giantswarm namespace.
	AppNamespace = "chart-operator.giantswarm.io/app-namespace"

	// Metadata annotation stores an app metadata URL from the appCatalog's index.yaml.
	Metadata = "application.giantswarm.io/metadata"
)
