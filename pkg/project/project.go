package project

var (
	description = "The app-operator manages apps in Kubernetes clusters."
	gitSHA      = "n/a"
	name        = "app-operator"
	source      = "https://github.com/giantswarm/app-operator"
	version     = "2.3.5"
)

// AppControlPlaneVersion is always 0.0.0 for control plane app CRs. These CRs
// are processed by app-operator-unique which always runs the latest version.
func AppControlPlaneVersion() string {
	return "0.0.0"
}

// AppTenantVersion is always 1.0.0 for tenant cluster app CRs using Helm 2.
// For app CRs using Helm 3 we use project.Version().
func AppTenantVersion() string {
	return "1.0.0"
}

func Description() string {
	return description
}

func GitSHA() string {
	return gitSHA
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
