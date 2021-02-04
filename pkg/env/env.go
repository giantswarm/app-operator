package env

import (
	"fmt"
	"os"
)

const (
	// EnvVarPodNamespace is the process environment variable representing the
	// POD_NAMESPACE env var which is populated via the Downward API.
	EnvVarPodNamespace = "POD_NAMESPACE"
)

var (
	podNamespace string
)

func init() {
	podNamespace = os.Getenv(EnvVarPodNamespace)
	if podNamespace == "" {
		panic(fmt.Sprintf("env var '%s' must not be empty", EnvVarPodNamespace))
	}
}

func PodNamespace() string {
	return podNamespace
}
