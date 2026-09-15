//go:build k8srequired
// +build k8srequired

package env

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	// EnvVarCircleCI is the process environment variable representing the
	// CIRCLECI env var.
	EnvVarCircleCI = "CIRCLECI"
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
	buildVersion  string
	circleCI      string
	circleBranch  string
	keepResources string
	kubeconfig    string
)

func init() {
	circleCI = os.Getenv(EnvVarCircleCI)
	keepResources = os.Getenv(EnvVarKeepResources)

	filePath := filepath.Join(os.Getenv("CIRCLE_WORKING_DIRECTORY"), ".build_version")
	buf, err := os.ReadFile(filePath)
	if err != nil {
		panic(fmt.Sprintf("error reading .build_version: %v", err))
	}

	buildVersion = strings.TrimSpace(string(buf))
	if buildVersion == "" {
		panic(".build_version must not be empty")
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

func BuildVersion() string {
	return buildVersion
}

func CircleCI() bool {
	return circleCI == strings.ToLower("true")
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
