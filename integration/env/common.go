//go:build k8srequired
// +build k8srequired

package env

import (
	"fmt"
	"os"
	"strings"
)

const (
	// EnvVarCircleCI is the process environment variable representing the
	// CIRCLECI env var.
	EnvVarCircleCI = "CIRCLECI"
	// EnvVarCircleSHA is the process environment variable representing the
	// CIRCLE_SHA1 env var.
	EnvVarCircleSHA = "CIRCLE_SHA1"
	//EnvVarCircleBranch is the branch the build is running against.
	EnvVarCircleBranch = "CIRCLE_BRANCH"
	// EnvVarE2EKubeconfig is the process environment variable representing the
	// E2E_KUBECONFIG env var.
	EnvVarE2EKubeconfig = "E2E_KUBECONFIG"
	// EnvVarKeepResources is the process environment variable representing the
	// KEEP_RESOURCES env var.
	EnvVarKeepResources = "KEEP_RESOURCES"
)

var (
	circleCI      string
	circleSHA     string
	circleBranch  string
	keepResources string
	kubeconfig    string
)

func init() {
	circleCI = os.Getenv(EnvVarCircleCI)
	keepResources = os.Getenv(EnvVarKeepResources)

	circleSHA = os.Getenv(EnvVarCircleSHA)
	if circleSHA == "" {
		panic(fmt.Sprintf("env var '%s' must not be empty", EnvVarCircleSHA))
	}

	circleBranch = os.Getenv(EnvVarCircleBranch)
	if circleBranch == "" {
		panic(fmt.Sprintf("env var '%s' must not be empty", EnvVarCircleBranch))
	}

	kubeconfig = os.Getenv(EnvVarE2EKubeconfig)
	if kubeconfig == "" {
		panic(fmt.Sprintf("env var '%s' must not be empty", EnvVarE2EKubeconfig))
	}
}

func CircleCI() bool {
	return circleCI == strings.ToLower("true")
}

func CircleSHA() string {
	return circleSHA
}

func CircleBranch() string {
	return circleBranch
}

func IsMainBranch() bool {
	return CircleBranch() == "master" || CircleBranch() == "main"
}

func KeepResources() bool {
	return keepResources == strings.ToLower("true")
}

func KubeConfigPath() string {
	return kubeconfig
}
