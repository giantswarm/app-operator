package cordonchart

import (
	"context"
	"fmt"
	"github.com/giantswarm/apiextensions/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/apiextensions/pkg/clientset/versioned/fake"
	"github.com/giantswarm/app-operator/pkg/annotation"
	"github.com/giantswarm/micrologger/microloggertest"
	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"reflect"
	"testing"

	"github.com/giantswarm/app-operator/service/controller/app/v1/controllercontext"
)

func Test_Resource_EnsureCreated(t *testing.T) {
	tests := []struct {
		name               string
		obj                *v1alpha1.App
		chart              *v1alpha1.Chart
		expectedAnnotation map[string]string
		errorMatcher       func(error) bool
	}{
		{
			name: "case 0: added cordon annotations",
			obj: &v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "prometheus",
					Namespace: "default",
					Annotations: map[string]string{
						annotation.CordonReason: "migrating to app",
						annotation.CordonUntil:  "2019-12-31T12:59:00",
					},
				},
			},
			chart: &v1alpha1.Chart{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Chart",
					APIVersion: "application.giantswarm.io",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "prometheus",
					Namespace: "default",
					Annotations: map[string]string{
						"md5-checksum": "ff011ab44",
					},
				},
			},
			expectedAnnotation: map[string]string{
				"md5-checksum":          "ff011ab44",
				annotation.CordonReason: "migrating to app",
				annotation.CordonUntil:  "2019-12-31T12:59:00",
			},
		},
		{
			name: "case 1: app is not cordoned",
			obj: &v1alpha1.App{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "prometheus",
					Namespace: "default",
				},
			},
			chart: &v1alpha1.Chart{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Chart",
					APIVersion: "application.giantswarm.io",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "prometheus",
					Namespace: "default",
					Annotations: map[string]string{
						"md5-checksum": "ff011ab44",
					},
				},
			},
			expectedAnnotation: map[string]string{
				"md5-checksum": "ff011ab44",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {

			var err error

			objs := make([]runtime.Object, 0, 0)
			if tc.obj != nil {
				objs = append(objs, tc.obj)
			}
			if tc.chart != nil {
				objs = append(objs, tc.chart)
			}

			g8sClient := fake.NewSimpleClientset(objs...)

			c := Config{
				Logger: microloggertest.New(),

				ChartNamespace: "default",
			}
			r, err := New(c)
			if err != nil {
				t.Fatalf("error == %#v, want nil", err)
			}

			ctlConfig := controllercontext.Context{
				G8sClient: g8sClient,
			}
			ctx := controllercontext.NewContext(context.Background(), ctlConfig)

			err = r.EnsureCreated(ctx, tc.obj)
			switch {
			case err != nil && tc.errorMatcher == nil:
				t.Fatalf("error == %#v, want nil", err)
			case err == nil && tc.errorMatcher != nil:
				t.Fatalf("error == nil, want non-nil")
			case err != nil && !tc.errorMatcher(err):
				t.Fatalf("error == %#v, want matching", err)
			}

			if err == nil && tc.errorMatcher == nil {
				chart, err := g8sClient.ApplicationV1alpha1().Charts(tc.chart.Namespace).Get(tc.chart.Name, metav1.GetOptions{})
				if err != nil {
					t.Fatalf("error == %#v, want nil", err)
				}
				if !reflect.DeepEqual(chart.GetAnnotations(), tc.expectedAnnotation) {
					fmt.Println(chart.GetAnnotations())
					t.Fatalf("want matching app.annotations \n %s", cmp.Diff(chart.GetAnnotations(), tc.expectedAnnotation))
				}
			}
		})
	}
}
