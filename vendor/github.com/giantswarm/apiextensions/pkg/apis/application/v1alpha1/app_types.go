package v1alpha1

import (
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	kindApp = "App"
)

// NewAppCRD returns a new custom resource definition for App.
// This might look something like the following.
//
//     apiVersion: apiextensions.k8s.io/v1beta1
//     kind: CustomResourceDefinition
//     metadata:
//       name: app.application.giantswarm.io
//     spec:
//       group: application.giantswarm.io
//       scope: Namespaced
//       version: v1alpha1
//       names:
//         kind: App
//         plural: apps
//         singular: app
//
func NewAppCRD() *apiextensionsv1beta1.CustomResourceDefinition {
	return &apiextensionsv1beta1.CustomResourceDefinition{
		TypeMeta: metav1.TypeMeta{
			APIVersion: apiextensionsv1beta1.SchemeGroupVersion.String(),
			Kind:       "CustomResourceDefinition",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "apps.application.giantswarm.io",
		},
		Spec: apiextensionsv1beta1.CustomResourceDefinitionSpec{
			Group:   "application.giantswarm.io",
			Scope:   "Namespaced",
			Version: "v1alpha1",
			Names: apiextensionsv1beta1.CustomResourceDefinitionNames{
				Kind:     "App",
				Plural:   "apps",
				Singular: "app",
			},
			Subresources: &apiextensionsv1beta1.CustomResourceSubresources{
				Status: &apiextensionsv1beta1.CustomResourceSubresourceStatus{},
			},
		},
	}
}

func NewAppTypeMeta() metav1.TypeMeta {
	return metav1.TypeMeta{
		APIVersion: version,
		Kind:       kindApp,
	}
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// App CR example as below.
//
//     apiVersion: application.giantswarm.io/v1alpha1
//     kind: App
//     metadata:
//       name: "My-Cool-Prometheus"
//       namespace: “12345”
//     spec:
//       name: “kubernetes-prometheus”
//       catalog: "giant-swarm"
//       namespace: “monitoring”
//       release: 1.0.0
//
//       kubeConfig:
//         secret:
//          name: "giantswarm-12345"
//          namespace: "12345"
//
//     status:
//       app: “DEPLOYED”
//
type App struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              AppSpec   `json:"spec"`
	Status            AppStatus `json:"status" yaml:"status"`
}

type AppSpec struct {
	// Name is the name of the application to be deployed.
	// e.g. kubernetes-prometheus
	Name string `json:"name" yaml:"name"`
	// Catalog is the name of the application catalog this app belongs to.
	// e.g. giant-swarm
	Catalog string `json:"catalog" yaml:"catalog"`
	// Namespace is the namespace where the application should be deployed.
	// e.g. monitoring
	Namespace string `json:"namespace" yaml:"namespace"`
	// Release is the version of this application which we would like to use.
	// e.g. 1.0.0
	Release string `json:"release" yaml:"release"`
	// KubeConfig is the name of secret resource  which represent specific k8s cluster where the application could be deployed.
	KubeConfig AppSpecKubeConfig `json:"kubeConfig" yaml:"kubeConfig"`
}

type AppSpecKubeConfig struct {
	Secret AppSpecKubeConfigSecret `json:"secret" yaml:"secret"`
}

type AppSpecKubeConfigSecret struct {
	Name      string `json:"name" yaml:"name"`
	Namespace string `json:"namespace" yaml:"namespace"`
}

type AppStatus struct {
	App string `json:"app" yaml:"app"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type AppList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []App `json:"items"`
}
