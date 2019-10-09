package utils

import (
	"testing"

	"k8s.io/apimachinery/pkg/api/equality"

	corev1 "k8s.io/api/core/v1"

	kanaryv1alpha1 "github.com/k8s-kanary/kanary/pkg/apis/kanary/v1alpha1"
	kanaryv1alpha1test "github.com/k8s-kanary/kanary/pkg/apis/kanary/v1alpha1/test"
	utilstest "github.com/k8s-kanary/kanary/pkg/controller/kanarystatefulset/utils/test"
)

func TestNewCanaryServiceForKanaryStatefulset(t *testing.T) {
	namespace := "kanary"
	name := "foo"
	dummyKD := kanaryv1alpha1test.NewKanaryStatefulset(name, namespace, name, 3, nil)

	type args struct {
		kd             *kanaryv1alpha1.KanaryStatefulset
		service        *corev1.Service
		overwriteLabel bool
	}
	tests := []struct {
		name string
		args args
		want *corev1.Service
	}{
		{
			name: "nodePort service",
			args: args{
				kd:             dummyKD,
				service:        utilstest.NewService(name, namespace, nil, &utilstest.NewServiceOptions{Type: corev1.ServiceTypeNodePort, Ports: []corev1.ServicePort{{Port: 8080, NodePort: 3010}}}),
				overwriteLabel: false,
			},
			want: utilstest.NewService(name+"-kanary-"+name, namespace, map[string]string{kanaryv1alpha1.KanaryStatefulsetActivateLabelKey: kanaryv1alpha1.KanaryStatefulsetLabelValueTrue, kanaryv1alpha1.KanaryStatefulsetKanaryNameLabelKey: name}, &utilstest.NewServiceOptions{Type: corev1.ServiceTypeClusterIP, Ports: []corev1.ServicePort{{Port: 8080}}}),
		},
		{
			name: "loadbalancer service",
			args: args{
				kd:             dummyKD,
				service:        utilstest.NewService(name, namespace, nil, &utilstest.NewServiceOptions{Type: corev1.ServiceTypeLoadBalancer}),
				overwriteLabel: false,
			},
			want: utilstest.NewService(name+"-kanary-"+name, namespace, map[string]string{kanaryv1alpha1.KanaryStatefulsetActivateLabelKey: kanaryv1alpha1.KanaryStatefulsetLabelValueTrue, kanaryv1alpha1.KanaryStatefulsetKanaryNameLabelKey: name}, &utilstest.NewServiceOptions{Type: corev1.ServiceTypeClusterIP}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, _ := NewCanaryServiceForKanaryStatefulset(tt.args.kd, tt.args.service, tt.args.overwriteLabel, PrepareSchemeForOwnerRef(), false); !equality.Semantic.DeepEqual(got, tt.want) {
				t.Errorf("NewCanaryServiceForKanaryStatefulset() = %v, want %v", got, tt.want)
			}
		})
	}
}
