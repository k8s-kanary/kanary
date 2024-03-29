package traffic

import (
	"reflect"
	"testing"
	"time"

	kanaryv1alpha1 "github.com/k8s-kanary/kanary/pkg/apis/kanary/v1alpha1"
	kanaryv1alpha1test "github.com/k8s-kanary/kanary/pkg/apis/kanary/v1alpha1/test"
	"github.com/k8s-kanary/kanary/pkg/controller/kanarystatefulset/utils"
	utilstest "github.com/k8s-kanary/kanary/pkg/controller/kanarystatefulset/utils/test"
	appsv1beta1 "k8s.io/api/apps/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

func Test_kanaryServiceImpl_Traffic(t *testing.T) {
	logf.SetLogger(logf.ZapLogger(true))
	log := logf.Log.WithName("Test_cleanupImpl_Traffic")

	var (
		name            = "foo"
		serviceName     = "foo"
		namespace       = "kanary"
		defaultReplicas = int32(5)

		kanaryServiceTraffic = &kanaryv1alpha1.KanaryStatefulsetSpecTraffic{
			Source: kanaryv1alpha1.KanaryServiceKanaryStatefulsetSpecTrafficSource,
		}
	)

	type args struct {
		kclient   client.Client
		kd        *kanaryv1alpha1.KanaryStatefulset
		canaryDep *appsv1beta1.Deployment
	}
	tests := []struct {
		name       string
		args       args
		wantResult reconcile.Result
		wantErr    bool
		wantFunc   func(kclient client.Client, kd *kanaryv1alpha1.KanaryStatefulset) error
	}{
		{
			name: "service is active, nothing change",
			args: args{
				kclient: fake.NewFakeClient([]runtime.Object{
					utilstest.NewService(serviceName, namespace, nil, nil),
					utilstest.NewService(serviceName+"-kanary-"+name, namespace, map[string]string{kanaryv1alpha1.KanaryStatefulsetKanaryNameLabelKey: name, kanaryv1alpha1.KanaryStatefulsetActivateLabelKey: kanaryv1alpha1.KanaryStatefulsetLabelValueTrue}, nil),
				}...),
				kd: kanaryv1alpha1test.NewKanaryStatefulset(name, namespace, serviceName, defaultReplicas, &kanaryv1alpha1test.NewKanaryStatefulsetOptions{Traffic: kanaryServiceTraffic}),
			},
			wantResult: reconcile.Result{},
			wantErr:    false,
		},
		{
			name: "service is active, create kanary service",
			args: args{
				kclient: fake.NewFakeClient([]runtime.Object{
					utilstest.NewService(serviceName, namespace, nil, nil),
				}...),
				kd: kanaryv1alpha1test.NewKanaryStatefulset(name, namespace, serviceName, defaultReplicas, &kanaryv1alpha1test.NewKanaryStatefulsetOptions{Traffic: kanaryServiceTraffic}),
			},
			wantResult: reconcile.Result{Requeue: true},
			wantErr:    false,
		},
		{
			name: "service no active, nothing todo",
			args: args{
				kclient: fake.NewFakeClient([]runtime.Object{
					utilstest.NewService(serviceName, namespace, nil, nil),
				}...),
				kd: kanaryv1alpha1test.NewKanaryStatefulset(name, namespace, serviceName, defaultReplicas, nil),
			},
			wantResult: reconcile.Result{Requeue: false},
			wantErr:    false,
		},
		{
			name: "service strategy is active, but service doesn't exist, return error",
			args: args{
				kclient: fake.NewFakeClient([]runtime.Object{}...),
				kd:      kanaryv1alpha1test.NewKanaryStatefulset(name, namespace, serviceName, defaultReplicas, &kanaryv1alpha1test.NewKanaryStatefulsetOptions{Traffic: kanaryServiceTraffic}),
			},
			wantResult: reconcile.Result{Requeue: true, RequeueAfter: 1 * time.Second},
			wantErr:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqLogger := log.WithValues("test:", tt.name)
			c := &kanaryServiceImpl{
				conf:   &tt.args.kd.Spec.Traffic,
				scheme: utils.PrepareSchemeForOwnerRef(),
			}
			_, gotResult, err := c.Traffic(tt.args.kclient, reqLogger, tt.args.kd, tt.args.canaryDep, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("kanaryServiceImpl.Traffic() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotResult, tt.wantResult) {
				t.Errorf("kanaryServiceImpl.Traffic() gotResult = %v, want %v", gotResult, tt.wantResult)
			}
			if tt.wantFunc != nil {
				if err = tt.wantFunc(tt.args.kclient, tt.args.kd); err != nil {
					t.Errorf("wantFunc returns an error: %v", err)
				}
			}
		})
	}
}

func Test_kanaryServiceImpl_Cleanup(t *testing.T) {
	logf.SetLogger(logf.ZapLogger(true))
	log := logf.Log.WithName("Test_cleanupImpl_Traffic")

	var (
		name            = "foo"
		serviceName     = "foo"
		namespace       = "kanary"
		defaultReplicas = int32(5)

		kanaryServiceTraffic = &kanaryv1alpha1.KanaryStatefulsetSpecTraffic{
			Source: kanaryv1alpha1.KanaryServiceKanaryStatefulsetSpecTrafficSource,
		}

		serviceTraffic = &kanaryv1alpha1.KanaryStatefulsetSpecTraffic{
			Source: kanaryv1alpha1.BothKanaryStatefulsetSpecTrafficSource,
		}

		noneTraffic = &kanaryv1alpha1.KanaryStatefulsetSpecTraffic{
			Source: kanaryv1alpha1.NoneKanaryStatefulsetSpecTrafficSource,
		}

		statusFailed = &kanaryv1alpha1.KanaryStatefulsetStatus{
			Conditions: []kanaryv1alpha1.KanaryStatefulsetCondition{
				{
					Type:   kanaryv1alpha1.FailedKanaryStatefulsetConditionType,
					Status: corev1.ConditionTrue,
				},
			},
		}
	)

	type args struct {
		kclient   client.Client
		kd        *kanaryv1alpha1.KanaryStatefulset
		canaryDep *appsv1beta1.Deployment
	}
	tests := []struct {
		name       string
		args       args
		wantStatus *kanaryv1alpha1.KanaryStatefulsetStatus
		wantResult reconcile.Result
		wantErr    bool
		wantFunc   func(kclient client.Client, kd *kanaryv1alpha1.KanaryStatefulset) error
	}{

		{
			name: "service not active, one service to clean",
			args: args{
				kclient: fake.NewFakeClient([]runtime.Object{
					utilstest.NewService(serviceName+"-kanary-"+name, namespace, map[string]string{kanaryv1alpha1.KanaryStatefulsetKanaryNameLabelKey: name}, nil),
				}...),
				kd: kanaryv1alpha1test.NewKanaryStatefulset(name, namespace, "", defaultReplicas, nil),
			},
			wantStatus: &kanaryv1alpha1.KanaryStatefulsetStatus{},
			wantResult: reconcile.Result{Requeue: true},
			wantErr:    false,
		},

		{
			name: "service not active, nothing to delete",
			args: args{
				kclient: fake.NewFakeClient([]runtime.Object{}...),
				kd:      kanaryv1alpha1test.NewKanaryStatefulset(name, namespace, "", defaultReplicas, nil),
			},
			wantStatus: &kanaryv1alpha1.KanaryStatefulsetStatus{},
			wantResult: reconcile.Result{},
			wantErr:    false,
		},
		{
			name: "service is active, nothing to delete",
			args: args{
				kclient: fake.NewFakeClient([]runtime.Object{
					utilstest.NewService(serviceName+"-kanary-"+name, namespace, map[string]string{kanaryv1alpha1.KanaryStatefulsetKanaryNameLabelKey: name}, nil),
				}...),
				kd: kanaryv1alpha1test.NewKanaryStatefulset(name, namespace, "", defaultReplicas, &kanaryv1alpha1test.NewKanaryStatefulsetOptions{Traffic: kanaryServiceTraffic}),
			},
			wantStatus: &kanaryv1alpha1.KanaryStatefulsetStatus{},
			wantResult: reconcile.Result{},
			wantErr:    false,
		},

		{
			name: "kd status failed, service not activated",
			args: args{
				kclient: fake.NewFakeClient([]runtime.Object{
					utilstest.NewService(serviceName, namespace, map[string]string{}, nil),
				}...),
				kd:        kanaryv1alpha1test.NewKanaryStatefulset(name, namespace, serviceName, defaultReplicas, &kanaryv1alpha1test.NewKanaryStatefulsetOptions{Traffic: noneTraffic, Status: statusFailed}),
				canaryDep: utilstest.NewDeployment(name+"-kanary-"+name, namespace, 1, nil),
			},
			wantStatus: statusFailed,
			wantResult: reconcile.Result{},
			wantErr:    false,
		},
		{
			name: "kd status failed, service activated, desactivate",
			args: args{
				kclient: fake.NewFakeClient([]runtime.Object{
					utilstest.NewService(serviceName, namespace, map[string]string{"app": "foo"}, nil),
					utilstest.NewService(serviceName+"-kanary-"+name, namespace, map[string]string{kanaryv1alpha1.KanaryStatefulsetKanaryNameLabelKey: name, "app": "foo"}, nil),
					utilstest.NewPod(name, namespace, "hash", &utilstest.NewPodOptions{Labels: map[string]string{kanaryv1alpha1.KanaryStatefulsetKanaryNameLabelKey: name, "app": "foo", "version": "v2"}}),
				}...),
				kd:        kanaryv1alpha1test.NewKanaryStatefulset(name, namespace, serviceName, defaultReplicas, &kanaryv1alpha1test.NewKanaryStatefulsetOptions{Traffic: serviceTraffic, Status: statusFailed}),
				canaryDep: utilstest.NewDeployment(name+"-kanary", namespace, 1, &utilstest.NewDeploymentOptions{Selector: map[string]string{kanaryv1alpha1.KanaryStatefulsetKanaryNameLabelKey: name, "app": "foo", "version": "v2"}}),
			},
			wantStatus: statusFailed,
			wantResult: reconcile.Result{Requeue: true},
			wantErr:    false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqLogger := log.WithValues("test:", tt.name)
			c := &kanaryServiceImpl{
				conf:   &tt.args.kd.Spec.Traffic,
				scheme: utils.PrepareSchemeForOwnerRef(),
			}
			gotStatus, gotResult, err := c.Cleanup(tt.args.kclient, reqLogger, tt.args.kd, tt.args.canaryDep, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("cleanupImpl.Traffic() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotStatus, tt.wantStatus) {
				t.Errorf("cleanupImpl.Traffic() gotStatus = %v, want %v", gotStatus, tt.wantStatus)
			}
			if !reflect.DeepEqual(gotResult, tt.wantResult) {
				t.Errorf("cleanupImpl.Traffic() gotResult = %v, want %v", gotResult, tt.wantResult)
			}
			if tt.wantFunc != nil {
				if err = tt.wantFunc(tt.args.kclient, tt.args.kd); err != nil {
					t.Errorf("wantFunc returns an error: %v", err)
				}
			}
		})
	}
}
