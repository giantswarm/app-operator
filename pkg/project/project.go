package project

var (
	description = "The app-operator manages apps in Kubernetes clusters."
	gitSHA      = "n/a"
	name        = "app-operator"
	source      = "https://github.com/giantswarm/app-operator"
	version     = "1.1.7"
)

// AppControlPlaneVersion is always 0.0.0 for control plane app CRs. These CRs
// are processed by app-operator-unique which always runs the latest version.
func AppControlPlaneVersion() string {
	return "0.0.0"
}

// AppTenantVersion is currently always 1.0.0 for tenant cluster app CRs. In a
// future release this hardcoding will be removed.
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
