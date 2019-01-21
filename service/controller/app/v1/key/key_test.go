package key

import (
	"reflect"
	"testing"

	"github.com/giantswarm/apiextensions/pkg/apis/application/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_AppName(t *testing.T) {
	expectedName := "giant-swarm-name"

	obj := v1alpha1.App{
		Spec: v1alpha1.AppSpec{
			Name: "giant-swarm-name",
		},
	}

	if AppName(obj) != expectedName {
		t.Fatalf("app name %s, want %s", AppName(obj), expectedName)
	}
}

func Test_CatalogName(t *testing.T) {
	expectedName := "giant-swarm-catalog-name"

	obj := v1alpha1.App{
		Spec: v1alpha1.AppSpec{
			Name:    "giant-swarm-name",
			Catalog: "giant-swarm-catalog-name",
		},
	}

	if CatalogName(obj) != expectedName {
		t.Fatalf("catalog name %s, want %s", CatalogName(obj), expectedName)
	}
}

func Test_ConfigMapName(t *testing.T) {
	expectedName := "giant-swarm-configmap-name"

	obj := v1alpha1.App{
		Spec: v1alpha1.AppSpec{
			Name:    "giant-swarm-name",
			Catalog: "giant-swarm-catalog-name",
			Config: v1alpha1.AppSpecConfig{
				ConfigMap: v1alpha1.AppSpecConfigConfigMap{
					Name: "giant-swarm-configmap-name",
				},
			},
		},
	}

	if ConfigMapName(obj) != expectedName {
		t.Fatalf("configMap name %s, want %s", ConfigMapName(obj), expectedName)
	}
}

func Test_ConfigMapNamespace(t *testing.T) {
	expectedName := "giant-swarm-configmap-namespace"

	obj := v1alpha1.App{
		Spec: v1alpha1.AppSpec{
			Name:    "giant-swarm-name",
			Catalog: "giant-swarm-catalog-name",
			Config: v1alpha1.AppSpecConfig{
				ConfigMap: v1alpha1.AppSpecConfigConfigMap{
					Name:      "giant-swarm-configmap-name",
					Namespace: "giant-swarm-configmap-namespace",
				},
			},
		},
	}

	if ConfigMapNamespace(obj) != expectedName {
		t.Fatalf("configMap namespace %s, want %s", ConfigMapNamespace(obj), expectedName)
	}
}

func Test_Namespace(t *testing.T) {
	expectedName := "giant-swarm-namespace"

	obj := v1alpha1.App{
		Spec: v1alpha1.AppSpec{
			Name:      "giant-swarm-name",
			Namespace: "giant-swarm-namespace",
		},
	}

	if Namespace(obj) != expectedName {
		t.Fatalf("app namespace %s, want %s", Namespace(obj), expectedName)
	}
}

func Test_KubeConfigSecretName(t *testing.T) {
	expectedName := "cluster-12345-kubeconfig"

	obj := v1alpha1.App{
		Spec: v1alpha1.AppSpec{
			KubeConfig: v1alpha1.AppSpecKubeConfig{
				Secret: v1alpha1.AppSpecKubeConfigSecret{
					Name:      "cluster-12345-kubeconfig",
					Namespace: "default",
				},
			},
		},
	}

	if KubeConfigSecretName(obj) != expectedName {
		t.Fatalf("app namespace %s, want %s", KubeConfigSecretName(obj), expectedName)
	}
}

func Test_KubeConfigSecretNamespace(t *testing.T) {
	expectedNamespace := "default"

	obj := v1alpha1.App{
		Spec: v1alpha1.AppSpec{
			KubeConfig: v1alpha1.AppSpecKubeConfig{
				Secret: v1alpha1.AppSpecKubeConfigSecret{
					Name:      "cluster-12345-kubeconfig",
					Namespace: "default",
				},
			},
		},
	}

	if KubeConfigSecretNamespace(obj) != expectedNamespace {
		t.Fatalf("app namespace %s, want %s", KubeConfigSecretNamespace(obj), expectedNamespace)
	}
}

func Test_ReleaseName(t *testing.T) {
	expectedName := "giant-swarm-release"

	obj := v1alpha1.App{
		Spec: v1alpha1.AppSpec{
			Name:      "giant-swarm-name",
			Namespace: "giant-swarm-namespace",
			Release:   "giant-swarm-release",
		},
	}

	if ReleaseName(obj) != expectedName {
		t.Fatalf("app release %s, want %s", ReleaseName(obj), expectedName)
	}
}

func Test_SecretName(t *testing.T) {
	expectedName := "giant-swarm-secret-name"

	obj := v1alpha1.App{
		Spec: v1alpha1.AppSpec{
			Name:    "giant-swarm-name",
			Catalog: "giant-swarm-catalog-name",
			Config: v1alpha1.AppSpecConfig{
				Secret: v1alpha1.AppSpecConfigSecret{
					Name: "giant-swarm-secret-name",
				},
			},
		},
	}

	if SecretName(obj) != expectedName {
		t.Fatalf("secret name %s, want %s", SecretName(obj), expectedName)
	}
}

func Test_SecretNamespace(t *testing.T) {
	expectedName := "giant-swarm-secret-namespace"

	obj := v1alpha1.App{
		Spec: v1alpha1.AppSpec{
			Name:    "giant-swarm-name",
			Catalog: "giant-swarm-catalog-name",
			Config: v1alpha1.AppSpecConfig{
				Secret: v1alpha1.AppSpecConfigSecret{
					Namespace: "giant-swarm-secret-namespace",
				},
			},
		},
	}

	if SecretNamespace(obj) != expectedName {
		t.Fatalf("secret namespace %s, want %s", SecretNamespace(obj), expectedName)
	}
}

func Test_ToCustomResource(t *testing.T) {
	testCases := []struct {
		name           string
		input          interface{}
		expectedObject v1alpha1.App
		errorMatcher   func(error) bool
	}{
		{
			name: "case 0: basic match",
			input: &v1alpha1.App{
				Spec: v1alpha1.AppSpec{
					Name:      "giant-swarm-name",
					Namespace: "giant-swarm-namespace",
					Release:   "giant-swarm-release",
				},
			},
			expectedObject: v1alpha1.App{
				Spec: v1alpha1.AppSpec{
					Name:      "giant-swarm-name",
					Namespace: "giant-swarm-namespace",
					Release:   "giant-swarm-release",
				},
			},
		},
		{
			name:         "case 1: wrong type",
			input:        &v1alpha1.AppCatalog{},
			errorMatcher: IsWrongTypeError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := ToCustomResource(tc.input)
			switch {
			case err != nil && tc.errorMatcher == nil:
				t.Fatalf("error == %#v, want nil", err)
			case err == nil && tc.errorMatcher != nil:
				t.Fatalf("error == nil, want non-nil")
			case err != nil && !tc.errorMatcher(err):
				t.Fatalf("error == %#v, want matching", err)
			}

			if !reflect.DeepEqual(result, tc.expectedObject) {
				t.Fatalf("Custom Object == %#v, want %#v", result, tc.expectedObject)
			}
		})
	}
}

func TestVersionBundleVersion(t *testing.T) {
	testCases := []struct {
		name           string
		input          v1alpha1.App
		expectedObject string
		errorMatcher   func(error) bool
	}{
		{
			name: "case 0: basic match",
			input: v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"giantswarm.io/version-bundle": "0.1.0",
					},
				},
			},
			expectedObject: "0.1.0",
		},
		{
			name: "case 1: can't find key",
			input: v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"giantswarm.io/version": "",
					},
				},
			},
			expectedObject: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := VersionBundleVersion(tc.input)

			if !reflect.DeepEqual(result, tc.expectedObject) {
				t.Fatalf("version == %#v, want %#v", result, tc.expectedObject)
			}
		})
	}
}
