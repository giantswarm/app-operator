package status

import v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

type Request struct {
	AppName      string  `json:"app_name"`
	AppNamespace string  `json:"app_namespace"`
	AppVersion   string  `json:"app_version"`
	AuthToken    string  `json:"auth_token"`
	LastDeployed v1.Time `json:"last_deployed"`
	Reason       string  `json:"reason"`
	Status       string  `json:"status"`
	Version      string  `json:"version"`
}
