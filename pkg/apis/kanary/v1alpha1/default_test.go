package v1alpha1

import (
	"reflect"
	"testing"
	"time"

	"k8s.io/api/autoscaling/v2beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestIsDefaultedKanaryStatefulset(t *testing.T) {

	tests := []struct {
		name string
		kd   *KanaryStatefulset
		want bool
	}{
		{
			name: "not defaulted",
			kd:   &KanaryStatefulset{},
			want: false,
		},
		{
			name: "is defaulted",
			kd: &KanaryStatefulset{
				Spec: KanaryStatefulsetSpec{
					Scale: KanaryStatefulsetSpecScale{
						Static: &KanaryStatefulsetSpecScaleStatic{
							Replicas: NewInt32(1),
						},
					},
					Traffic: KanaryStatefulsetSpecTraffic{
						Source: ServiceKanaryStatefulsetSpecTrafficSource,
					},
					Validations: KanaryStatefulsetSpecValidationList{
						ValidationPeriod: &metav1.Duration{
							Duration: 15 * time.Minute,
						},
						InitialDelay: &metav1.Duration{
							Duration: 5 * time.Minute,
						},
						MaxIntervalPeriod: &metav1.Duration{
							Duration: 5 * time.Minute,
						},
						Items: []KanaryStatefulsetSpecValidation{
							{
								Manual: &KanaryStatefulsetSpecValidationManual{
									StatusAfterDealine: NoneKanaryStatefulsetSpecValidationManualDeadineStatus,
								},
							},
						},
					},
				},
			},
			want: true,
		},
		{
			name: "is defaulted",
			kd: &KanaryStatefulset{
				Spec: KanaryStatefulsetSpec{
					Scale: KanaryStatefulsetSpecScale{
						Static: &KanaryStatefulsetSpecScaleStatic{
							Replicas: NewInt32(1),
						},
					},
					Traffic: KanaryStatefulsetSpecTraffic{
						Source: ServiceKanaryStatefulsetSpecTrafficSource,
					},
					Validations: KanaryStatefulsetSpecValidationList{
						ValidationPeriod: &metav1.Duration{
							Duration: 15 * time.Minute,
						},
						InitialDelay: &metav1.Duration{
							Duration: 5 * time.Minute,
						},
						Items: []KanaryStatefulsetSpecValidation{
							{
								PromQL: &KanaryStatefulsetSpecValidationPromQL{
									PrometheusService:        "s",
									PodNameKey:               "pod",
									ContinuousValueDeviation: &ContinuousValueDeviation{},
								},
							},
						},
					},
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsDefaultedKanaryStatefulset(tt.kd); got != tt.want {
				t.Errorf("IsDefaultedKanaryStatefulset() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDefaultKanaryStatefulset(t *testing.T) {

	tests := []struct {
		name string
		kd   *KanaryStatefulset
		want *KanaryStatefulset
	}{
		{
			name: "not defaulted",
			kd: &KanaryStatefulset{
				Spec: KanaryStatefulsetSpec{},
			},
			want: &KanaryStatefulset{
				Spec: KanaryStatefulsetSpec{
					Scale: KanaryStatefulsetSpecScale{
						Static: &KanaryStatefulsetSpecScaleStatic{
							Replicas: NewInt32(1),
						},
					},
					Traffic: KanaryStatefulsetSpecTraffic{
						Source: NoneKanaryStatefulsetSpecTrafficSource,
					},
					Validations: KanaryStatefulsetSpecValidationList{
						ValidationPeriod: &metav1.Duration{
							Duration: 15 * time.Minute,
						},
						InitialDelay: &metav1.Duration{
							Duration: 0 * time.Minute,
						},
						MaxIntervalPeriod: &metav1.Duration{
							Duration: 20 * time.Second,
						},
						Items: []KanaryStatefulsetSpecValidation{
							{
								Manual: &KanaryStatefulsetSpecValidationManual{
									StatusAfterDealine: NoneKanaryStatefulsetSpecValidationManualDeadineStatus,
								},
							},
						},
					},
				},
			},
		},

		{
			name: "already some configuration",
			kd: &KanaryStatefulset{
				Spec: KanaryStatefulsetSpec{
					Scale: KanaryStatefulsetSpecScale{
						Static: &KanaryStatefulsetSpecScaleStatic{
							Replicas: NewInt32(1),
						},
					},
					Traffic: KanaryStatefulsetSpecTraffic{
						Source: KanaryServiceKanaryStatefulsetSpecTrafficSource,
					},
					Validations: KanaryStatefulsetSpecValidationList{
						ValidationPeriod: &metav1.Duration{
							Duration: 30 * time.Minute,
						},
						InitialDelay: &metav1.Duration{
							Duration: 5 * time.Minute,
						},
						MaxIntervalPeriod: &metav1.Duration{
							Duration: 5 * time.Minute,
						},
						Items: []KanaryStatefulsetSpecValidation{
							{
								PromQL: &KanaryStatefulsetSpecValidationPromQL{
									Query:                    "foo",
									ContinuousValueDeviation: &ContinuousValueDeviation{},
								},
							},
						},
					},
				},
			},
			want: &KanaryStatefulset{
				Spec: KanaryStatefulsetSpec{
					Scale: KanaryStatefulsetSpecScale{
						Static: &KanaryStatefulsetSpecScaleStatic{
							Replicas: NewInt32(1),
						},
					},
					Traffic: KanaryStatefulsetSpecTraffic{
						Source: KanaryServiceKanaryStatefulsetSpecTrafficSource,
					},
					Validations: KanaryStatefulsetSpecValidationList{
						ValidationPeriod: &metav1.Duration{
							Duration: 30 * time.Minute,
						},
						InitialDelay: &metav1.Duration{
							Duration: 5 * time.Minute,
						},
						MaxIntervalPeriod: &metav1.Duration{
							Duration: 5 * time.Minute,
						},
						Items: []KanaryStatefulsetSpecValidation{
							{
								PromQL: &KanaryStatefulsetSpecValidationPromQL{
									PrometheusService: "prometheus:9090",
									Query:             "foo",
									PodNameKey:        "pod",
									ContinuousValueDeviation: &ContinuousValueDeviation{
										MaxDeviationPercent: NewFloat64(10),
									},
								},
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DefaultKanaryStatefulset(tt.kd); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DefaultKanaryStatefulset() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsDefaultedKanaryStatefulsetSpecScale(t *testing.T) {
	type args struct {
		scale *KanaryStatefulsetSpecScale
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "already defaulted with static",
			args: args{
				scale: &KanaryStatefulsetSpecScale{
					Static: &KanaryStatefulsetSpecScaleStatic{
						Replicas: NewInt32(1),
					},
				},
			},
			want: true,
		},
		{
			name: "not defaulted at all",
			args: args{
				scale: &KanaryStatefulsetSpecScale{},
			},
			want: false,
		},
		{
			name: "HPA not defaulted (minReplicas, maxReplicas)",
			args: args{
				scale: &KanaryStatefulsetSpecScale{
					HPA: &HorizontalPodAutoscalerSpec{},
				},
			},
			want: false,
		},
		{
			name: "HPA not defaulted (Metrics)",
			args: args{
				scale: &KanaryStatefulsetSpecScale{
					HPA: &HorizontalPodAutoscalerSpec{
						MinReplicas: NewInt32(1),
						MaxReplicas: int32(5),
					},
				},
			},
			want: false,
		},
		{
			name: "HPA not defaulted (Metrics empty slice)",
			args: args{
				scale: &KanaryStatefulsetSpecScale{
					HPA: &HorizontalPodAutoscalerSpec{
						MinReplicas: NewInt32(1),
						MaxReplicas: int32(5),
						Metrics:     []v2beta1.MetricSpec{},
					},
				},
			},
			want: false,
		},
		{
			name: "HPA defaulted ",
			args: args{
				scale: &KanaryStatefulsetSpecScale{
					HPA: &HorizontalPodAutoscalerSpec{
						MinReplicas: NewInt32(1),
						MaxReplicas: int32(5),
						Metrics:     []v2beta1.MetricSpec{{}},
					},
				},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsDefaultedKanaryStatefulsetSpecScale(tt.args.scale); got != tt.want {
				t.Errorf("IsDefaultedKanaryStatefulsetSpecScale() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_defaultKanaryStatefulsetSpecValidationList(t *testing.T) {
	tests := []struct {
		name string
		list *KanaryStatefulsetSpecValidationList
		want *KanaryStatefulsetSpecValidationList
	}{
		{
			name: "nil list",
			list: &KanaryStatefulsetSpecValidationList{},
			want: &KanaryStatefulsetSpecValidationList{
				ValidationPeriod: &metav1.Duration{
					Duration: 15 * time.Minute,
				},
				InitialDelay: &metav1.Duration{
					Duration: 0 * time.Minute,
				},
				MaxIntervalPeriod: &metav1.Duration{
					Duration: 20 * time.Second,
				},
				Items: []KanaryStatefulsetSpecValidation{
					{
						Manual: &KanaryStatefulsetSpecValidationManual{
							StatusAfterDealine: NoneKanaryStatefulsetSpecValidationManualDeadineStatus,
						},
					},
				},
			},
		},
		{
			name: "one element not defaulted",
			list: &KanaryStatefulsetSpecValidationList{
				Items: []KanaryStatefulsetSpecValidation{{}},
			},
			want: &KanaryStatefulsetSpecValidationList{
				ValidationPeriod: &metav1.Duration{
					Duration: 15 * time.Minute,
				},
				InitialDelay: &metav1.Duration{
					Duration: 0 * time.Minute,
				},
				MaxIntervalPeriod: &metav1.Duration{
					Duration: 20 * time.Second,
				},
				Items: []KanaryStatefulsetSpecValidation{
					{
						Manual: &KanaryStatefulsetSpecValidationManual{
							StatusAfterDealine: NoneKanaryStatefulsetSpecValidationManualDeadineStatus,
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defaultKanaryStatefulsetSpecValidationList(tt.list)
			if !reflect.DeepEqual(tt.list, tt.want) {
				t.Errorf("defaultKanaryStatefulsetSpecValidationList() = %#v, want %#v", tt.list, tt.want)
			}
		})
	}
}
