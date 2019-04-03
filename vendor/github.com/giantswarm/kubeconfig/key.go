package kubeconfig

import (
	"github.com/giantswarm/apiextensions/pkg/apis/application/v1alpha1"
)

func inCluster(app v1alpha1.App) bool {
	return app.Spec.KubeConfig.InCluster
}

func secretName(app v1alpha1.App) string {
	return app.Spec.KubeConfig.Secret.Name
}

func secretNamespace(app v1alpha1.App) string {
	return app.Spec.KubeConfig.Secret.Namespace
}
