package validation

import (
	"context"
	"strings"
	"testing"

	"github.com/giantswarm/apiextensions/v3/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/apiextensions/v3/pkg/clientset/versioned/fake"
	"github.com/giantswarm/micrologger/microloggertest"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgofake "k8s.io/client-go/kubernetes/fake"
)

func Test_Resource_GetDesiredState(t *testing.T) {
	tests := []struct {
		name        string
		obj         v1alpha1.App
		configMaps  []*corev1.ConfigMap
		secrets     []*corev1.Secret
		expectedErr string
	}{
		{
			name: "case 0: flawless flow",
			obj: v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-cool-prometheus",
				},
				Spec: v1alpha1.AppSpec{
					Name:      "prometheus",
					Namespace: "monitoring",
					Version:   "1.0.0",
					Config: v1alpha1.AppSpecConfig{
						ConfigMap: v1alpha1.AppSpecConfigConfigMap{
							Name:      "giant-swarm-config",
							Namespace: "giantswarm",
						},
						Secret: v1alpha1.AppSpecConfigSecret{
							Name:      "giant-swarm-config",
							Namespace: "giantswarm",
						},
					},
					KubeConfig: v1alpha1.AppSpecKubeConfig{
						InCluster: true,
					},
				},
			},
			configMaps: []*corev1.ConfigMap{
				{
					Data: map[string]string{
						"values": "cluster: yaml\n",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "giant-swarm-config",
						Namespace: "giantswarm",
					},
				},
			},
			secrets: []*corev1.Secret{
				{
					Data: map[string][]byte{
						"values": []byte("cluster: yaml\n"),
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "giant-swarm-config",
						Namespace: "giantswarm",
					},
				},
			},
		},
		{
			name: "case 2: configmap not found",
			obj: v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-cool-prometheus",
				},
				Spec: v1alpha1.AppSpec{
					Config: v1alpha1.AppSpecConfig{
						ConfigMap: v1alpha1.AppSpecConfigConfigMap{
							Name:      "giant-swarm-config",
							Namespace: "giantswarm",
						},
					},
				},
			},
			expectedErr: "configmap `giant-swarm-config` in namespace `giantswarm` not found",
		},
		{
			name: "case 3: no namespace specified for configmap",
			obj: v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-cool-prometheus",
				},
				Spec: v1alpha1.AppSpec{
					Config: v1alpha1.AppSpecConfig{
						ConfigMap: v1alpha1.AppSpecConfigConfigMap{
							Name:      "giant-swarm-config",
							Namespace: "",
						},
					},
				},
			},
			expectedErr: "namespace is not specified for configmap `giant-swarm-config`",
		},
		{
			name: "case 4: secret not found",
			obj: v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-cool-prometheus",
				},
				Spec: v1alpha1.AppSpec{
					Config: v1alpha1.AppSpecConfig{
						Secret: v1alpha1.AppSpecConfigSecret{
							Name:      "giant-swarm-config",
							Namespace: "giantswarm",
						},
					},
				},
			},
			expectedErr: "secret `giant-swarm-config` in namespace `giantswarm` not found",
		},
		{
			name: "case 5: no namespace specified for secret",
			obj: v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-cool-prometheus",
				},
				Spec: v1alpha1.AppSpec{
					Config: v1alpha1.AppSpecConfig{
						Secret: v1alpha1.AppSpecConfigSecret{
							Name:      "giant-swarm-config",
							Namespace: "",
						},
					},
				},
			},
			expectedErr: "namespace is not specified for secret `giant-swarm-config`",
		},
		{
			name: "case 6: user configmap not found",
			obj: v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-cool-prometheus",
				},
				Spec: v1alpha1.AppSpec{
					UserConfig: v1alpha1.AppSpecUserConfig{
						ConfigMap: v1alpha1.AppSpecUserConfigConfigMap{
							Name:      "giant-swarm-config",
							Namespace: "giantswarm",
						},
					},
				},
			},
			expectedErr: "configmap `giant-swarm-config` in namespace `giantswarm` not found",
		},
		{
			name: "case 7: no namespace specified for user configmap",
			obj: v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-cool-prometheus",
				},
				Spec: v1alpha1.AppSpec{
					UserConfig: v1alpha1.AppSpecUserConfig{
						ConfigMap: v1alpha1.AppSpecUserConfigConfigMap{
							Name:      "giant-swarm-config",
							Namespace: "",
						},
					},
				},
			},
			expectedErr: "namespace is not specified for configmap `giant-swarm-config`",
		},
		{
			name: "case 8: user secret not found",
			obj: v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-cool-prometheus",
				},
				Spec: v1alpha1.AppSpec{
					UserConfig: v1alpha1.AppSpecUserConfig{
						Secret: v1alpha1.AppSpecUserConfigSecret{
							Name:      "giant-swarm-config",
							Namespace: "giantswarm",
						},
					},
				},
			},
			expectedErr: "secret `giant-swarm-config` in namespace `giantswarm` not found",
		},
		{
			name: "case 9: no namespace specified for user secret",
			obj: v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-cool-prometheus",
				},
				Spec: v1alpha1.AppSpec{
					UserConfig: v1alpha1.AppSpecUserConfig{
						Secret: v1alpha1.AppSpecUserConfigSecret{
							Name:      "giant-swarm-config",
							Namespace: "",
						},
					},
				},
			},
			expectedErr: "namespace is not specified for secret `giant-swarm-config`",
		},
		{
			name: "case 10: kubeconig secret not found",
			obj: v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-cool-prometheus",
				},
				Spec: v1alpha1.AppSpec{
					KubeConfig: v1alpha1.AppSpecKubeConfig{
						InCluster: false,
						Secret: v1alpha1.AppSpecKubeConfigSecret{
							Name:      "kubeconfig",
							Namespace: "giantswarm",
						},
					},
				},
			},
			expectedErr: "kubeconfig secret `kubeconfig` in namespace `giantswarm` not found",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			objs := make([]runtime.Object, 0)
			for _, cm := range tc.configMaps {
				objs = append(objs, cm)
			}

			for _, secret := range tc.secrets {
				objs = append(objs, secret)
			}

			c := Config{
				G8sClient: fake.NewSimpleClientset(),
				K8sClient: clientgofake.NewSimpleClientset(objs...),
				Logger:    microloggertest.New(),
			}
			r, err := New(c)
			if err != nil {
				t.Fatalf("error == %#v, want nil", err)
			}

			err = r.validateApp(context.TODO(), tc.obj)
			switch {
			case err != nil && tc.expectedErr == "":
				t.Fatalf("error == %#v, want nil", err)
			case err == nil && tc.expectedErr != "":
				t.Fatalf("error == nil, want non-nil")
			}

			if err != nil && tc.expectedErr != "" {
				if !strings.Contains(err.Error(), tc.expectedErr) {
					t.Fatalf("error == %#v, want %#v ", err.Error(), tc.expectedErr)
				}

			}
		})
	}
}
