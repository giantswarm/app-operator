package helmrelease

import (
	"reflect"
	"testing"
	"time"

	helmv2 "github.com/fluxcd/helm-controller/api/v2beta1"
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

	type suspendState struct {
		current  bool
		desired  bool
		expected bool
	}

	tests := []struct {
		name    string
		args    args
		suspend suspendState
	}{
		{
			name: "Pause timestamp annotation is kept",
			args: args{
				current: map[string]string{
					"app-operator.giantswarm.io/pause-ts":     tenminutesago,
					"app-operator.giantswarm.io/pause-reason": "foobar",
				},
				desired: map[string]string{
					"app-operator.giantswarm.io/pause-reason": "changed",
					"app-operator.giantswarm.io/pause-ts":     now,
				},
				result: map[string]string{
					"app-operator.giantswarm.io/pause-ts":     tenminutesago,
					"app-operator.giantswarm.io/pause-reason": "changed",
				},
			},
			suspend: suspendState{
				current:  true,
				desired:  true,
				expected: true,
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
			suspend: suspendState{
				current:  true,
				desired:  true,
				expected: true,
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
			suspend: suspendState{
				current:  true,
				desired:  false,
				expected: false,
			},
		},
		{
			name: "Pause annotations removed when timeout elapsed",
			args: args{
				current: map[string]string{
					"app-operator.giantswarm.io/pause-ts":     expired,
					"app-operator.giantswarm.io/pause-reason": "foobar",
				},
				desired: map[string]string{
					"app-operator.giantswarm.io/pause-ts":     now,
					"app-operator.giantswarm.io/pause-reason": "foobar",
				},
				result: map[string]string{},
			},
			suspend: suspendState{
				current:  true,
				desired:  true,
				expected: false,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			desired := getChart(tt.args.desired, tt.suspend.desired)
			r := &Resource{
				dependencyWaitTimeoutMinutes: 30,
			}

			r.configurePause(getChart(tt.args.current, tt.suspend.current), desired)
			if !reflect.DeepEqual(desired.Annotations, tt.args.result) {
				t.Logf("Wanted %v, got %v", tt.args.result, desired.Annotations)
				t.Fail()
			}

			if desired.Spec.Suspend != tt.suspend.expected {
				t.Logf("Wanted %t, got %t", tt.suspend.expected, desired.Spec.Suspend)
				t.Fail()
			}
		})
	}
}

func getChart(annotations map[string]string, suspend bool) *helmv2.HelmRelease {
	return &helmv2.HelmRelease{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: annotations,
		},
		Spec: helmv2.HelmReleaseSpec{
			Suspend: suspend,
		},
	}
}
