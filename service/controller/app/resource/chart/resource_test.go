package chart

import (
	"reflect"
	"testing"
	"time"

	"github.com/giantswarm/apiextensions-application/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_copyAnnotations(t *testing.T) {

	now := time.Now().Format(time.RFC3339)
	tenminutesago := time.Now().Add(-10 * time.Minute).Format(time.RFC3339)
	expired := time.Now().Add(-1 * time.Hour).Format(time.RFC3339)

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
			name: "Pause annotation is kept when not handled by app operator",
			args: args{
				current: map[string]string{
					"chart-operator.giantswarm.io/paused": "true",
				},
				desired: map[string]string{},
				result: map[string]string{
					"chart-operator.giantswarm.io/paused": "true",
				},
			},
		},
		{
			name: "Pause annotation is deleted when handled by app operator",
			args: args{
				current: map[string]string{
					"chart-operator.giantswarm.io/paused":     "true",
					"app-operator.giantswarm.io/pause-reason": "foobar",
				},
				desired: map[string]string{},
				result:  map[string]string{},
			},
		},
		{
			name: "Pause timestamp annotation is kept",
			args: args{
				current: map[string]string{
					"chart-operator.giantswarm.io/paused":     "true",
					"app-operator.giantswarm.io/pause-ts":     tenminutesago,
					"app-operator.giantswarm.io/pause-reason": "foobar",
				},
				desired: map[string]string{
					"chart-operator.giantswarm.io/paused":     "true",
					"app-operator.giantswarm.io/pause-reason": "changed",
					"app-operator.giantswarm.io/pause-ts":     now,
				},
				result: map[string]string{
					"app-operator.giantswarm.io/pause-ts":     tenminutesago,
					"app-operator.giantswarm.io/pause-reason": "changed",
					"chart-operator.giantswarm.io/paused":     "true",
				},
			},
		},
		{
			name: "Pause timestamp annotation is unchanged",
			args: args{
				current: map[string]string{
					"chart-operator.giantswarm.io/paused":     "true",
					"app-operator.giantswarm.io/pause-ts":     tenminutesago,
					"app-operator.giantswarm.io/pause-reason": "foobar",
				},
				desired: map[string]string{
					"chart-operator.giantswarm.io/paused":     "true",
					"app-operator.giantswarm.io/pause-ts":     now,
					"app-operator.giantswarm.io/pause-reason": "foobar",
				},
				result: map[string]string{
					"chart-operator.giantswarm.io/paused":     "true",
					"app-operator.giantswarm.io/pause-ts":     tenminutesago,
					"app-operator.giantswarm.io/pause-reason": "foobar",
				},
			},
		},
		{
			name: "Pause timestamp annotation is deleted when pause is removed",
			args: args{
				current: map[string]string{
					"app-operator.giantswarm.io/pause-ts": "foobar",
				},
				desired: map[string]string{},
				result:  map[string]string{},
			},
		},
		{
			name: "Pause annotations removed when timeout elapsed",
			args: args{
				current: map[string]string{
					"chart-operator.giantswarm.io/paused":     "true",
					"app-operator.giantswarm.io/pause-ts":     expired,
					"app-operator.giantswarm.io/pause-reason": "foobar",
				},
				desired: map[string]string{
					"chart-operator.giantswarm.io/paused":     "true",
					"app-operator.giantswarm.io/pause-ts":     now,
					"app-operator.giantswarm.io/pause-reason": "foobar",
				},
				result: map[string]string{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			desired := getChart(tt.args.desired)
			r := &Resource{
				dependencyWaitTimeoutMinutes: 30,
			}

			r.copyAnnotations(getChart(tt.args.current), desired)
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
