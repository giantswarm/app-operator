package key

import (
	"reflect"
	"testing"

	"github.com/giantswarm/apiextensions/pkg/apis/application/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_AppConfigMapName(t *testing.T) {
	expectedName := "giant-swarm-configmap-name"

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

	if AppConfigMapName(obj) != expectedName {
		t.Fatalf("AppConfigMapName %#q, want %#q", AppConfigMapName(obj), expectedName)
	}
}

func Test_AppConfigMapNamespace(t *testing.T) {
	expectedNamespace := "giant-swarm-configmap-namespace"

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

	if AppConfigMapNamespace(obj) != expectedNamespace {
		t.Fatalf("AppConfigMapNamespace %#q, want %#q", AppConfigMapNamespace(obj), expectedNamespace)
	}
}

func Test_AppName(t *testing.T) {
	expectedName := "giant-swarm-name"

	obj := v1alpha1.App{
		Spec: v1alpha1.AppSpec{
			Name: "giant-swarm-name",
		},
	}

	if AppName(obj) != expectedName {
		t.Fatalf("app name %#q, want %#q", AppName(obj), expectedName)
	}
}
func Test_AppSecretName(t *testing.T) {
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

	if AppSecretName(obj) != expectedName {
		t.Fatalf("AppSecretName %#q, want %#q", AppSecretName(obj), expectedName)
	}
}

func Test_AppSecretNamespace(t *testing.T) {
	expectedNamespace := "giant-swarm-secret-namespace"

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

	if AppSecretNamespace(obj) != expectedNamespace {
		t.Fatalf("AppSecretNamespace %#q, want %#q", AppSecretNamespace(obj), expectedNamespace)
	}
}

func Test_AppStatus(t *testing.T) {
	expectedStatus := v1alpha1.AppStatus{
		AppVersion: "0.12.0",
		Release: v1alpha1.AppStatusRelease{
			Status: "DEPLOYED",
		},
		Version: "0.1.0",
	}

	obj := v1alpha1.App{
		Status: v1alpha1.AppStatus{
			AppVersion: "0.12.0",
			Release: v1alpha1.AppStatusRelease{
				Status: "DEPLOYED",
			},
			Version: "0.1.0",
		},
	}

	if AppStatus(obj) != expectedStatus {
		t.Fatalf("app status %#q, want %#q", AppStatus(obj), expectedStatus)
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
		t.Fatalf("catalog name %#q, want %#q", CatalogName(obj), expectedName)
	}
}

func Test_ChartStatus(t *testing.T) {
	expectedStatus := v1alpha1.ChartStatus{
		AppVersion: "0.12.0",
		Release: v1alpha1.ChartStatusRelease{
			Status: "DEPLOYED",
		},
		Version: "0.1.0",
	}

	obj := v1alpha1.Chart{
		Status: v1alpha1.ChartStatus{
			AppVersion: "0.12.0",
			Release: v1alpha1.ChartStatusRelease{
				Status: "DEPLOYED",
			},
			Version: "0.1.0",
		},
	}

	if ChartStatus(obj) != expectedStatus {
		t.Fatalf("chart status %#q, want %#q", ChartStatus(obj), expectedStatus)
	}
}

func Test_ChartConfigMapName(t *testing.T) {
	expectedName := "my-test-app-chart-values"

	obj := v1alpha1.App{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-test-app",
			Namespace: "giantswarn",
		},
		Spec: v1alpha1.AppSpec{
			Name:    "test-app",
			Catalog: "test-catalog",
			Config: v1alpha1.AppSpecConfig{
				ConfigMap: v1alpha1.AppSpecConfigConfigMap{
					Name: "test-app-value",
				},
			},
		},
	}

	if ChartConfigMapName(obj) != expectedName {
		t.Fatalf("chartConfigMapName %#q, want %#q", ChartConfigMapName(obj), expectedName)
	}
}

func Test_InCluster(t *testing.T) {
	obj := v1alpha1.App{
		Spec: v1alpha1.AppSpec{
			KubeConfig: v1alpha1.AppSpecKubeConfig{
				InCluster: true,
			},
		},
	}

	if !InCluster(obj) {
		t.Fatalf("app namespace %#v, want %#v", InCluster(obj), true)
	}
}

func Test_KubecConfigFinalizer(t *testing.T) {
	obj := v1alpha1.App{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-test-app",
		},
	}

	if KubeConfigFinalizer(obj) != "app-operator.giantswarm.io/app-my-test-app" {
		t.Fatalf("kubeconfig finalizer name %#v, want %#v", KubeConfigFinalizer(obj), "app-operator.giantswarm.io/app-my-test-app")
	}
}

func Test_KubecConfigSecretName(t *testing.T) {
	obj := v1alpha1.App{
		Spec: v1alpha1.AppSpec{
			KubeConfig: v1alpha1.AppSpecKubeConfig{
				InCluster: false,
				Secret: v1alpha1.AppSpecKubeConfigSecret{
					Name: "kubename",
				},
			},
		},
	}

	if KubecConfigSecretName(obj) != "kubename" {
		t.Fatalf("kubeconfig secret name %#v, want %#v", KubecConfigSecretName(obj), "kubename")
	}
}

func Test_KubecConfigSecretNamespace(t *testing.T) {
	obj := v1alpha1.App{
		Spec: v1alpha1.AppSpec{
			KubeConfig: v1alpha1.AppSpecKubeConfig{
				InCluster: false,
				Secret: v1alpha1.AppSpecKubeConfigSecret{
					Namespace: "kubenamespace",
				},
			},
		},
	}

	if KubecConfigSecretNamespace(obj) != "kubenamespace" {
		t.Fatalf("kubeconfig secret namespace %#v, want %#v", KubecConfigSecretNamespace(obj), "kubenamespace")
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
		t.Fatalf("app namespace %#q, want %#q", Namespace(obj), expectedName)
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
					Version:   "1.2.3",
				},
			},
			expectedObject: v1alpha1.App{
				Spec: v1alpha1.AppSpec{
					Name:      "giant-swarm-name",
					Namespace: "giant-swarm-namespace",
					Version:   "1.2.3",
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

func Test_UserConfigMapName(t *testing.T) {
	expectedName := "giant-swarm-user-configmap-name"

	obj := v1alpha1.App{
		Spec: v1alpha1.AppSpec{
			Name:    "giant-swarm-name",
			Catalog: "giant-swarm-catalog-name",
			UserConfig: v1alpha1.AppSpecUserConfig{
				ConfigMap: v1alpha1.AppSpecUserConfigConfigMap{
					Name: "giant-swarm-user-configmap-name",
				},
			},
		},
	}

	if UserConfigMapName(obj) != expectedName {
		t.Fatalf("UserConfigMapName %#q, want %#q", UserConfigMapName(obj), expectedName)
	}
}

func Test_UserConfigMapNamespace(t *testing.T) {
	expectedNamespace := "giant-swarm-user-configmap-namespace"

	obj := v1alpha1.App{
		Spec: v1alpha1.AppSpec{
			Name:    "giant-swarm-name",
			Catalog: "giant-swarm-catalog-name",
			UserConfig: v1alpha1.AppSpecUserConfig{
				ConfigMap: v1alpha1.AppSpecUserConfigConfigMap{
					Namespace: "giant-swarm-user-configmap-namespace",
				},
			},
		},
	}

	if UserConfigMapNamespace(obj) != expectedNamespace {
		t.Fatalf("UserConfigMapNamespace %#q, want %#q", UserConfigMapNamespace(obj), expectedNamespace)
	}
}

func Test_UserSecretName(t *testing.T) {
	expectedName := "giant-swarm-user-secret-name"

	obj := v1alpha1.App{
		Spec: v1alpha1.AppSpec{
			Name:    "giant-swarm-name",
			Catalog: "giant-swarm-catalog-name",
			UserConfig: v1alpha1.AppSpecUserConfig{
				Secret: v1alpha1.AppSpecUserConfigSecret{
					Name: "giant-swarm-user-secret-name",
				},
			},
		},
	}

	if UserSecretName(obj) != expectedName {
		t.Fatalf("UserSecretName %#q, want %#q", UserSecretName(obj), expectedName)
	}
}

func Test_UserSecretNamespace(t *testing.T) {
	expectedNamespace := "giant-swarm-user-secret-namespace"

	obj := v1alpha1.App{
		Spec: v1alpha1.AppSpec{
			Name:    "giant-swarm-name",
			Catalog: "giant-swarm-catalog-name",
			UserConfig: v1alpha1.AppSpecUserConfig{
				Secret: v1alpha1.AppSpecUserConfigSecret{
					Namespace: "giant-swarm-user-secret-namespace",
				},
			},
		},
	}

	if UserSecretNamespace(obj) != expectedNamespace {
		t.Fatalf("UserSecretNamespace %#q, want %#q", UserSecretNamespace(obj), expectedNamespace)
	}
}

func Test_Version(t *testing.T) {
	expectedVersion := "1.2.3"

	obj := v1alpha1.App{
		Spec: v1alpha1.AppSpec{
			Name:      "prometheus",
			Namespace: "monitoring",
			Version:   "1.2.3",
		},
	}

	if Version(obj) != expectedVersion {
		t.Fatalf("app version %#q, want %#q", Version(obj), expectedVersion)
	}
}

func Test_VersionLabel(t *testing.T) {
	testCases := []struct {
		name            string
		obj             v1alpha1.App
		expectedVersion string
		errorMatcher    func(error) bool
	}{
		{
			name: "case 0: basic match",
			obj: v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app-operator.giantswarm.io/version": "1.0.0",
					},
				},
			},
			expectedVersion: "1.0.0",
		},
		{
			name: "case 1: different value",
			obj: v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app-operator.giantswarm.io/version": "2.0.0",
					},
				},
			},
			expectedVersion: "2.0.0",
		},
		{
			name: "case 2: incorrect label",
			obj: v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"chart-operator.giantswarm.io/version": "1.0.0",
					},
				},
			},
			expectedVersion: "",
		},
		{
			name:            "case 3: no labels",
			obj:             v1alpha1.App{},
			expectedVersion: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := VersionLabel(tc.obj)

			if !reflect.DeepEqual(result, tc.expectedVersion) {
				t.Fatalf("Version label == %#v, want %#v", result, tc.expectedVersion)
			}
		})
	}
}
