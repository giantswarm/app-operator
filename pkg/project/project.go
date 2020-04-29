package project

var (
	description = "The app-operator manages apps in Kubernetes clusters."
	gitSHA      = "n/a"
	name        = "app-operator"
	source      = "https://github.com/giantswarm/app-operator"
	version     = "1.0.2-dev"
)

// AppVersion is fixed for app CRs. Its version is not linked to a release.
// We may revisit this in future.
func AppVersion() string {
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
