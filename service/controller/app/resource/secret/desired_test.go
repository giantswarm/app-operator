package secret

import (
	"context"
	"reflect"
	"strconv"
	"testing"

	"github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/app/v7/pkg/values"
	"github.com/giantswarm/k8sclient/v7/pkg/k8sclienttest"
	"github.com/giantswarm/k8smetadata/pkg/annotation"
	"github.com/giantswarm/k8smetadata/pkg/label"
	"github.com/giantswarm/micrologger/microloggertest"
	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgofake "k8s.io/client-go/kubernetes/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/giantswarm/app-operator/v6/service/controller/app/controllercontext"
)

func Test_Resource_GetDesiredState(t *testing.T) {
	testCases := []struct {
		name               string
		obj                *v1alpha1.App
		catalog            v1alpha1.Catalog
		secrets            []*corev1.Secret
		expectedSecret     *corev1.Secret
		expectedUserConfig *v1alpha1.AppSpecUserConfig
		errorMatcher       func(error) bool
	}{
		{
			name: "case 0: secret is nil when there are no secrets",
			obj: &v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-test-app",
					Namespace: "giantswarm",
				},
				Spec: v1alpha1.AppSpec{
					Catalog:   "app-catalog",
					Name:      "test-app",
					Namespace: "kube-system",
				},
			},
			catalog: v1alpha1.Catalog{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-catalog",
				},
			},
			expectedSecret:     nil,
			expectedUserConfig: &v1alpha1.AppSpecUserConfig{},
		},
		{
			name: "case 1: basic match with app secrets",
			obj: &v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-prometheus",
					Namespace: "giantswarm",
				},
				Spec: v1alpha1.AppSpec{
					Catalog:   "app-catalog",
					Name:      "prometheus",
					Namespace: "monitoring",
					Config: v1alpha1.AppSpecConfig{
						Secret: v1alpha1.AppSpecConfigSecret{
							Name:      "test-cluster-secrets",
							Namespace: "giantswarm",
						},
					},
				},
			},
			catalog: v1alpha1.Catalog{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-catalog",
				},
			},
			secrets: []*corev1.Secret{
				{
					Data: map[string][]byte{
						"secrets": []byte("cluster: yaml\n"),
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-cluster-secrets",
						Namespace: "giantswarm",
					},
				},
			},
			expectedSecret: &corev1.Secret{
				Data: map[string][]byte{
					"values": []byte("cluster: yaml\n"),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-prometheus-chart-secrets",
					Namespace: "giantswarm",
					Annotations: map[string]string{
						annotation.Notes: "DO NOT EDIT. Values managed by app-operator.",
					},
					Labels: map[string]string{
						label.ManagedBy: "app-operator",
					},
				},
			},
			expectedUserConfig: &v1alpha1.AppSpecUserConfig{},
		},
		{
			name: "case 2: user-values configmap",
			obj: &v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-app",
					Namespace: "giantswarm",
				},
				Spec: v1alpha1.AppSpec{
					Catalog:   "app-catalog",
					Name:      "test-app",
					Namespace: "kube-system",
				},
			},
			catalog: v1alpha1.Catalog{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-catalog",
				},
			},
			secrets: []*corev1.Secret{
				{
					Data: map[string][]byte{
						"values": []byte("cluster: yaml\n"),
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-app-user-secrets",
						Namespace: "giantswarm",
					},
				},
			},
			expectedSecret: &corev1.Secret{
				Data: map[string][]byte{
					"values": []byte("cluster: yaml\n"),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-app-chart-secrets",
					Namespace: "giantswarm",
					Annotations: map[string]string{
						annotation.Notes: "DO NOT EDIT. Values managed by app-operator.",
					},
					Labels: map[string]string{
						label.ManagedBy: "app-operator",
					},
				},
			},
			expectedUserConfig: &v1alpha1.AppSpecUserConfig{
				Secret: v1alpha1.AppSpecUserConfigSecret{
					Name:      "test-app-user-secrets",
					Namespace: "giantswarm",
				},
			},
		},
		{
			name: "case 3: user provided secrets",
			obj: &v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-app",
					Namespace: "giantswarm",
				},
				Spec: v1alpha1.AppSpec{
					Catalog:   "app-catalog",
					Name:      "test-app",
					Namespace: "kube-system",
					UserConfig: v1alpha1.AppSpecUserConfig{
						Secret: v1alpha1.AppSpecUserConfigSecret{
							Name:      "custom-secrets",
							Namespace: "giantswarm",
						},
					},
				},
			},
			catalog: v1alpha1.Catalog{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-catalog",
				},
			},
			secrets: []*corev1.Secret{
				{
					Data: map[string][]byte{
						"values": []byte("cluster: yaml\n"),
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "custom-secrets",
						Namespace: "giantswarm",
					},
				},
			},
			expectedSecret: &corev1.Secret{
				Data: map[string][]byte{
					"values": []byte("cluster: yaml\n"),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-app-chart-secrets",
					Namespace: "giantswarm",
					Annotations: map[string]string{
						annotation.Notes: "DO NOT EDIT. Values managed by app-operator.",
					},
					Labels: map[string]string{
						label.ManagedBy: "app-operator",
					},
				},
			},
			expectedUserConfig: &v1alpha1.AppSpecUserConfig{
				Secret: v1alpha1.AppSpecUserConfigSecret{
					Name:      "custom-secrets",
					Namespace: "giantswarm",
				},
			},
		},
		{
			name: "case 4: user provided secrets over default named secret",
			obj: &v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-app",
					Namespace: "giantswarm",
				},
				Spec: v1alpha1.AppSpec{
					Catalog:   "app-catalog",
					Name:      "test-app",
					Namespace: "kube-system",
					UserConfig: v1alpha1.AppSpecUserConfig{
						Secret: v1alpha1.AppSpecUserConfigSecret{
							Name:      "custom-secrets",
							Namespace: "giantswarm",
						},
					},
				},
			},
			catalog: v1alpha1.Catalog{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-catalog",
				},
			},
			secrets: []*corev1.Secret{
				{
					Data: map[string][]byte{
						"values": []byte("default: name\n"),
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-app-user-secrets",
						Namespace: "giantswarm",
					},
				},
				{
					Data: map[string][]byte{
						"values": []byte("cluster: yaml\n"),
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "custom-secrets",
						Namespace: "giantswarm",
					},
				},
			},
			expectedSecret: &corev1.Secret{
				Data: map[string][]byte{
					"values": []byte("cluster: yaml\n"),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-app-chart-secrets",
					Namespace: "giantswarm",
					Annotations: map[string]string{
						annotation.Notes: "DO NOT EDIT. Values managed by app-operator.",
					},
					Labels: map[string]string{
						label.ManagedBy: "app-operator",
					},
				},
			},
			expectedUserConfig: &v1alpha1.AppSpecUserConfig{
				Secret: v1alpha1.AppSpecUserConfigSecret{
					Name:      "custom-secrets",
					Namespace: "giantswarm",
				},
			},
		},
	}

	var err error

	for i, tc := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			s := runtime.NewScheme()
			s.AddKnownTypes(v1alpha1.SchemeGroupVersion, &v1alpha1.App{})

			objs := make([]runtime.Object, 0)
			for _, cm := range tc.secrets {
				objs = append(objs, cm)
			}

			k8sClient := clientgofake.NewSimpleClientset(objs...)
			ctrlClient := fake.NewClientBuilder().WithScheme(s).WithObjects(tc.obj).Build()

			var ctx context.Context
			{
				c := controllercontext.Context{
					Clients: controllercontext.Clients{
						K8s: k8sclienttest.NewClients(k8sclienttest.ClientsConfig{
							CtrlClient: ctrlClient,
							K8sClient:  k8sClient,
						}),
					},
					Catalog: tc.catalog,
				}
				ctx = controllercontext.NewContext(context.Background(), c)
			}

			var valuesService *values.Values
			{
				c := values.Config{
					K8sClient: k8sClient,
					Logger:    microloggertest.New(),
				}

				valuesService, err = values.New(c)
				if err != nil {
					t.Fatalf("error == %#v, want nil", err)
				}
			}

			c := Config{
				Logger: microloggertest.New(),
				Values: valuesService,

				ChartNamespace: "giantswarm",
			}
			r, err := New(c)
			if err != nil {
				t.Fatalf("error == %#v, want nil", err)
			}

			result, err := r.GetDesiredState(ctx, tc.obj)
			switch {
			case err != nil && tc.errorMatcher == nil:
				t.Fatalf("error == %#v, want nil", err)
			case err == nil && tc.errorMatcher != nil:
				t.Fatalf("error == nil, want non-nil")
			case err != nil && !tc.errorMatcher(err):
				t.Fatalf("error == %#v, want matching", err)
			}

			if result != nil && tc.expectedSecret == nil {
				t.Fatalf("expected nil secret got %#v", result)
			}
			if result == nil && tc.expectedSecret != nil {
				t.Fatal("expected non-nil secret got nil")
			}

			if tc.expectedSecret != nil {
				secret, err := toSecret(result)
				if err != nil {
					t.Fatalf("error == %#v, want nil", err)
				}

				if !reflect.DeepEqual(secret.ObjectMeta, tc.expectedSecret.ObjectMeta) {
					t.Fatalf("want matching objectmeta \n %s", cmp.Diff(secret.ObjectMeta, tc.expectedSecret.ObjectMeta))
				}
				if !reflect.DeepEqual(secret.Data, tc.expectedSecret.Data) {
					data := toStringMap(secret.Data)
					expectedData := toStringMap(tc.expectedSecret.Data)

					t.Fatalf("want matching data \n %s", cmp.Diff(data, expectedData))
				}
			}

			if tc.expectedUserConfig != nil {
				_ = ctrlClient.Get(ctx, types.NamespacedName{Name: tc.obj.GetName(), Namespace: tc.obj.GetNamespace()}, tc.obj)
				if !reflect.DeepEqual(&tc.obj.Spec.UserConfig, tc.expectedUserConfig) {
					t.Fatalf("want matching userconfig \n %s", cmp.Diff(&tc.obj.Spec.UserConfig, tc.expectedUserConfig))
				}
			}
		})
	}
}

func toStringMap(input map[string][]byte) map[string]string {
	result := map[string]string{}

	for k, v := range input {
		result[k] = string(v)
	}

	return result
}
