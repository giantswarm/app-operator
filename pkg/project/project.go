package project

var (
	description = "The app-operator manages apps in Kubernetes clusters."
	gitSHA      = "n/a"
	name        = "app-operator"
	source      = "https://github.com/giantswarm/app-operator"
	version     = "6.4.4"
)

func Description() string {
	return description
}

func GitSHA() string {
	return gitSHA
}

// ManagementClusterAppVersion is always 0.0.0 for management cluster app CRs. These CRs
// are processed by app-operator-unique which always runs the latest version.
func ManagementClusterAppVersion() string {
	return "0.0.0"
}

func Name() string {
	return name
}

func Source() string {
	return source
}

func Version() string {
	return version
}
