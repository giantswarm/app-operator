package index

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/giantswarm/apiextensions/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/micrologger/microloggertest"
	"github.com/google/go-cmp/cmp"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfake "k8s.io/client-go/kubernetes/fake"
)

func Test_Resource_GetDesiredState(t *testing.T) {
	tests := []struct {
		name              string
		obj               *v1alpha1.AppCatalog
		expectedConfigMap *v1.ConfigMap
		h                 func(w http.ResponseWriter, r *http.Request)
		errorMatcher      func(error) bool
	}{
		{
			name: "case 0: flawless flow",
			obj: &v1alpha1.AppCatalog{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "giantswarm",
					Namespace: "default",
					Labels: map[string]string{
						"app-operator.giantswarm.io/version": "1.0.0",
					},
				},
				Spec: v1alpha1.AppCatalogSpec{
					Title:       "Giant Swarm",
					Description: "Catalog of Apps by Giant Swarm",
					Storage: v1alpha1.AppCatalogSpecStorage{
						Type: "helm",
						URL:  "https://giantswarm.github.com/app-catalog/",
					},
					LogoURL: "https://s.giantswarm.io/...",
				},
			},
			expectedConfigMap: &v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "giantswarm",
					Namespace: "default",
					Labels: map[string]string{
						"giantswarm.io/managed-by": "app-operator",
					},
				},
				Data: map[string]string{
					"index.yaml": "test yaml",
				},
			},
			h: func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte("test yaml"))
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(tc.h))
			defer ts.Close()

			tc.obj.Spec.Storage.URL = ts.URL

			c := Config{
				K8sClient: k8sfake.NewSimpleClientset(),
				Logger:    microloggertest.New(),

				ProjectName: "app-operator",
			}
			r, err := New(c)
			if err != nil {
				t.Fatalf("error == %#v, want nil", err)
			}

			result, err := r.GetDesiredState(context.Background(), tc.obj)
			switch {
			case err != nil && tc.errorMatcher == nil:
				t.Fatalf("error == %#v, want nil", err)
			case err == nil && tc.errorMatcher != nil:
				t.Fatalf("error == nil, want non-nil")
			case err != nil && !tc.errorMatcher(err):
				t.Fatalf("error == %#v, want matching", err)
			}

			if err == nil && tc.errorMatcher == nil {
				configMap, err := toConfigMap(result)
				if err != nil {
					t.Fatalf("error == %#v, want nil", err)
				}

				if !reflect.DeepEqual(configMap.ObjectMeta, tc.expectedConfigMap.ObjectMeta) {
					t.Fatalf("want matching objectmeta \n %s", cmp.Diff(configMap.ObjectMeta, tc.expectedConfigMap.ObjectMeta))
				}
				if !reflect.DeepEqual(configMap.Data, tc.expectedConfigMap.Data) {
					t.Fatalf("want matching data \n %s", cmp.Diff(configMap.Data, tc.expectedConfigMap.Data))
				}
				if !reflect.DeepEqual(configMap.TypeMeta, tc.expectedConfigMap.TypeMeta) {
					t.Fatalf("want matching typemeta \n %s", cmp.Diff(configMap.TypeMeta, tc.expectedConfigMap.TypeMeta))
				}
			}
		})
	}
}
