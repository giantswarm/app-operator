package chart

import (
	"reflect"
	"testing"

	"github.com/giantswarm/apiextensions-application/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_copyAnnotations(t *testing.T) {
	type args struct {
		current map[string]string
		desired map[string]string
		result  map[string]string
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "Add new annotation",
			args: args{
				current: map[string]string{},
				desired: map[string]string{
					"chart-operator.giantswarm.io/foo": "foobar",
				},
				result: map[string]string{
					"chart-operator.giantswarm.io/foo": "foobar",
				},
			},
		},
		{
			name: "Change existing annotation",
			args: args{
				current: map[string]string{
					"chart-operator.giantswarm.io/foo": "foo",
				},
				desired: map[string]string{
					"chart-operator.giantswarm.io/foo": "foobar",
				},
				result: map[string]string{
					"chart-operator.giantswarm.io/foo": "foobar",
				},
			},
		},
		{
			name: "Deleting existing annotation",
			args: args{
				current: map[string]string{
					"chart-operator.giantswarm.io/foo": "foo",
				},
				desired: map[string]string{
					"chart-operator.giantswarm.io/foo-": "",
				},
				result: map[string]string{},
			},
		},
		{
			name: "Deleting not owned annotation",
			args: args{
				current: map[string]string{
					"foobar": "foo",
				},
				desired: map[string]string{},
				result:  map[string]string{},
			},
		},
		{
			name: "Attempting to delete annotation that's not there",
			args: args{
				current: map[string]string{
					"chart-operator.giantswarm.io/foo": "foo",
				},
				desired: map[string]string{
					"chart-operator.giantswarm.io/bar-": "",
				},
				result: map[string]string{
					"chart-operator.giantswarm.io/foo": "foo",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			desired := getChart(tt.args.desired)
			copyAnnotations(getChart(tt.args.current), desired)
			if !reflect.DeepEqual(desired.Annotations, tt.args.result) {
				t.Logf("Wanted %v, got %v", tt.args.result, desired.Annotations)
				t.Fail()
			}
		})
	}
}

func getChart(annotations map[string]string) *v1alpha1.Chart {
	return &v1alpha1.Chart{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: annotations,
		},
	}
}
