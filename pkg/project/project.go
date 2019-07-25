package project

var (
	description = "The app-operator manages apps in Kubernetes clusters."
	name        = "app-operator"
	gitSHA      = "n/a"
	source      = "https://github.com/giantswarm/app-operator"
	version     = "n/a"
)

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
