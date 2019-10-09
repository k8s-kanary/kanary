package utils

import (
	"reflect"
	"testing"

	kanaryv1alpha1 "github.com/k8s-kanary/kanary/pkg/apis/kanary/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

func TestIsKanaryStatefulsetFailed(t *testing.T) {
	type args struct {
		status *kanaryv1alpha1.KanaryStatefulsetStatus
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "failed",
			args: args{
				status: &kanaryv1alpha1.KanaryStatefulsetStatus{
					Conditions: []kanaryv1alpha1.KanaryStatefulsetCondition{
						{
							Type:   kanaryv1alpha1.FailedKanaryStatefulsetConditionType,
							Status: corev1.ConditionTrue,
						},
					},
				},
			},
			want: true,
		},
		{
			name: "not failed",
			args: args{
				status: &kanaryv1alpha1.KanaryStatefulsetStatus{},
			},
			want: false,
		},
		{
			name: "not failed, conditionFalse",
			args: args{
				status: &kanaryv1alpha1.KanaryStatefulsetStatus{
					Conditions: []kanaryv1alpha1.KanaryStatefulsetCondition{
						{
							Type:   kanaryv1alpha1.FailedKanaryStatefulsetConditionType,
							Status: corev1.ConditionFalse,
						},
					},
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsKanaryStatefulsetFailed(tt.args.status); got != tt.want {
				t.Errorf("IsKanaryStatefulsetFailed() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsKanaryStatefulsetSucceeded(t *testing.T) {
	type args struct {
		status *kanaryv1alpha1.KanaryStatefulsetStatus
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "succeed",
			args: args{
				status: &kanaryv1alpha1.KanaryStatefulsetStatus{
					Conditions: []kanaryv1alpha1.KanaryStatefulsetCondition{
						{
							Type:   kanaryv1alpha1.SucceededKanaryStatefulsetConditionType,
							Status: corev1.ConditionTrue,
						},
					},
				},
			},
			want: true,
		},
		{
			name: "not succeed",
			args: args{
				status: &kanaryv1alpha1.KanaryStatefulsetStatus{},
			},
			want: false,
		},
		{
			name: "not succeed, conditionFalse",
			args: args{
				status: &kanaryv1alpha1.KanaryStatefulsetStatus{
					Conditions: []kanaryv1alpha1.KanaryStatefulsetCondition{
						{
							Type:   kanaryv1alpha1.SucceededKanaryStatefulsetConditionType,
							Status: corev1.ConditionFalse,
						},
					},
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsKanaryStatefulsetSucceeded(tt.args.status); got != tt.want {
				t.Errorf("IsKanaryStatefulsetSucceeded() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_updateStatusWithReport(t *testing.T) {
	type args struct {
		kd     *kanaryv1alpha1.KanaryStatefulset
		status *kanaryv1alpha1.KanaryStatefulsetStatus
	}
	tests := []struct {
		name string
		args args
		want *kanaryv1alpha1.KanaryStatefulsetStatus
	}{
		{
			name: "default report",
			args: args{
				kd: &kanaryv1alpha1.KanaryStatefulset{
					Spec: kanaryv1alpha1.KanaryStatefulsetSpec{
						Traffic: kanaryv1alpha1.KanaryStatefulsetSpecTraffic{},
					},
				},
				status: &kanaryv1alpha1.KanaryStatefulsetStatus{
					Conditions: []kanaryv1alpha1.KanaryStatefulsetCondition{
						kanaryv1alpha1.KanaryStatefulsetCondition{
							Status: corev1.ConditionTrue,
							Type:   kanaryv1alpha1.ScheduledKanaryStatefulsetConditionType,
						},
					},

					Report: kanaryv1alpha1.KanaryStatefulsetStatusReport{},
				},
			},
			want: &kanaryv1alpha1.KanaryStatefulsetStatus{
				Report: kanaryv1alpha1.KanaryStatefulsetStatusReport{
					Status:     string(kanaryv1alpha1.ScheduledKanaryStatefulsetConditionType),
					Scale:      "static",
					Validation: "unknow",
				},
			},
		},
		{
			name: "promQL validation",
			args: args{
				kd: &kanaryv1alpha1.KanaryStatefulset{
					Spec: kanaryv1alpha1.KanaryStatefulsetSpec{
						Traffic: kanaryv1alpha1.KanaryStatefulsetSpecTraffic{
							Mirror: &kanaryv1alpha1.KanaryStatefulsetSpecTrafficMirror{},
						},
						Validations: kanaryv1alpha1.KanaryStatefulsetSpecValidationList{
							Items: []kanaryv1alpha1.KanaryStatefulsetSpecValidation{
								{PromQL: &kanaryv1alpha1.KanaryStatefulsetSpecValidationPromQL{}},
							},
						},
					},
				},
				status: &kanaryv1alpha1.KanaryStatefulsetStatus{
					Conditions: []kanaryv1alpha1.KanaryStatefulsetCondition{
						kanaryv1alpha1.KanaryStatefulsetCondition{
							Status: corev1.ConditionTrue,
							Type:   kanaryv1alpha1.ScheduledKanaryStatefulsetConditionType,
						},
						kanaryv1alpha1.KanaryStatefulsetCondition{
							Status: corev1.ConditionTrue,
							Type:   kanaryv1alpha1.RunningKanaryStatefulsetConditionType,
						},
					},
					Report: kanaryv1alpha1.KanaryStatefulsetStatusReport{},
				},
			},
			want: &kanaryv1alpha1.KanaryStatefulsetStatus{
				Report: kanaryv1alpha1.KanaryStatefulsetStatusReport{
					Status:     string(kanaryv1alpha1.RunningKanaryStatefulsetConditionType),
					Scale:      "static",
					Validation: "promQL",
				},
			},
		},
		{
			name: "labelWatch validation",
			args: args{
				kd: &kanaryv1alpha1.KanaryStatefulset{
					Spec: kanaryv1alpha1.KanaryStatefulsetSpec{
						Traffic: kanaryv1alpha1.KanaryStatefulsetSpecTraffic{
							Mirror: &kanaryv1alpha1.KanaryStatefulsetSpecTrafficMirror{},
						},
						Validations: kanaryv1alpha1.KanaryStatefulsetSpecValidationList{
							Items: []kanaryv1alpha1.KanaryStatefulsetSpecValidation{
								{LabelWatch: &kanaryv1alpha1.KanaryStatefulsetSpecValidationLabelWatch{}},
							},
						},
					},
				},
				status: &kanaryv1alpha1.KanaryStatefulsetStatus{
					Conditions: []kanaryv1alpha1.KanaryStatefulsetCondition{
						kanaryv1alpha1.KanaryStatefulsetCondition{
							Status: corev1.ConditionTrue,
							Type:   kanaryv1alpha1.ScheduledKanaryStatefulsetConditionType,
						},
						kanaryv1alpha1.KanaryStatefulsetCondition{
							Status: corev1.ConditionTrue,
							Type:   kanaryv1alpha1.RunningKanaryStatefulsetConditionType,
						},
						kanaryv1alpha1.KanaryStatefulsetCondition{
							Status: corev1.ConditionTrue,
							Type:   kanaryv1alpha1.FailedKanaryStatefulsetConditionType,
						},
					},
					Report: kanaryv1alpha1.KanaryStatefulsetStatusReport{},
				},
			},
			want: &kanaryv1alpha1.KanaryStatefulsetStatus{
				Report: kanaryv1alpha1.KanaryStatefulsetStatusReport{
					Status:     string(kanaryv1alpha1.FailedKanaryStatefulsetConditionType),
					Scale:      "static",
					Validation: "labelWatch",
				},
			},
		},
		{
			name: "manual validation",
			args: args{
				kd: &kanaryv1alpha1.KanaryStatefulset{
					Spec: kanaryv1alpha1.KanaryStatefulsetSpec{
						Traffic: kanaryv1alpha1.KanaryStatefulsetSpecTraffic{
							Mirror: &kanaryv1alpha1.KanaryStatefulsetSpecTrafficMirror{},
						},
						Validations: kanaryv1alpha1.KanaryStatefulsetSpecValidationList{
							Items: []kanaryv1alpha1.KanaryStatefulsetSpecValidation{
								{Manual: &kanaryv1alpha1.KanaryStatefulsetSpecValidationManual{}},
							},
						},
					},
				},
				status: &kanaryv1alpha1.KanaryStatefulsetStatus{
					Conditions: []kanaryv1alpha1.KanaryStatefulsetCondition{
						kanaryv1alpha1.KanaryStatefulsetCondition{
							Status: corev1.ConditionTrue,
							Type:   kanaryv1alpha1.ScheduledKanaryStatefulsetConditionType,
						},
						kanaryv1alpha1.KanaryStatefulsetCondition{
							Status: corev1.ConditionTrue,
							Type:   kanaryv1alpha1.RunningKanaryStatefulsetConditionType,
						},
						kanaryv1alpha1.KanaryStatefulsetCondition{
							Status: corev1.ConditionTrue,
							Type:   kanaryv1alpha1.SucceededKanaryStatefulsetConditionType,
						},
					},
					Report: kanaryv1alpha1.KanaryStatefulsetStatusReport{},
				},
			},
			want: &kanaryv1alpha1.KanaryStatefulsetStatus{
				Report: kanaryv1alpha1.KanaryStatefulsetStatusReport{
					Status:     string(kanaryv1alpha1.SucceededKanaryStatefulsetConditionType),
					Scale:      "static",
					Validation: "manual",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if updateStatusReport(tt.args.kd, tt.args.status); !reflect.DeepEqual(tt.args.status.Report, tt.want.Report) {
				t.Errorf("updateStatusWithReport() = %v, want %v", tt.args.status, tt.want)
			}
		})
	}
}
