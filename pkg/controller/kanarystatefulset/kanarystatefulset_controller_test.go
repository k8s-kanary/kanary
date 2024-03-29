package kanarystatefulset

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	appsv1beta1 "k8s.io/api/apps/v1beta1"
	corev1 "k8s.io/api/core/v1"

	"k8s.io/client-go/kubernetes/scheme"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"

	kanaryv1alpha1 "github.com/k8s-kanary/kanary/pkg/apis/kanary/v1alpha1"
	kanaryv1alpha1test "github.com/k8s-kanary/kanary/pkg/apis/kanary/v1alpha1/test"
	utilstest "github.com/k8s-kanary/kanary/pkg/controller/kanarystatefulset/utils/test"
)

func TestReconcileKanaryStatefulset_Reconcile(t *testing.T) {
	logf.SetLogger(logf.ZapLogger(true))

	var (
		name            = "foo"
		serviceName     = "foo"
		namespace       = "kanary"
		defaultReplicas = int32(5)

		kanaryServiceTraffic = &kanaryv1alpha1.KanaryStatefulsetSpecTraffic{
			Source: kanaryv1alpha1.KanaryServiceKanaryStatefulsetSpecTrafficSource,
		}
	)

	// Register operator types with the runtime scheme.
	s := scheme.Scheme
	s.AddKnownTypes(kanaryv1alpha1.SchemeGroupVersion, &kanaryv1alpha1.KanaryStatefulset{})

	type fields struct {
		client client.Client
		scheme *runtime.Scheme
	}
	tests := []struct {
		name     string
		fields   fields
		request  reconcile.Request
		want     reconcile.Result
		wantErr  bool
		wantFunc func(*ReconcileKanaryStatefulset) error
	}{
		{
			name: "[INIT] KanaryStatefulset dont exist",

			request: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      name,
					Namespace: namespace,
				},
			},
			fields: fields{
				scheme: s,
				client: fake.NewFakeClient([]runtime.Object{}...),
			},
			want: reconcile.Result{
				Requeue: false,
			},
		},

		{
			name: "[INIT] KanaryStatefulset Not defaulted",

			request: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      name,
					Namespace: namespace,
				},
			},
			fields: fields{
				scheme: s,
				client: fake.NewFakeClient([]runtime.Object{
					kanaryv1alpha1test.NewKanaryStatefulset(name, namespace, "", defaultReplicas, nil),
				}...),
			},
			want: reconcile.Result{
				Requeue: true,
			},
			wantFunc: func(r *ReconcileKanaryStatefulset) error {
				kd := &kanaryv1alpha1.KanaryStatefulset{}
				err := r.client.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: namespace}, kd)
				if err != nil && errors.IsNotFound(err) {
					return fmt.Errorf("unable to get the created deployment, %v", err)
				}

				if kd.Spec.Scale.Static == nil || kd.Spec.Scale.Static.Replicas == nil {
					return fmt.Errorf("kd.Spec.Scale.Static.Replicas should be defaulted")
				}

				return err
			},
		},

		{
			name: "[INIT] canary Deployment creation",

			request: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      name,
					Namespace: namespace,
				},
			},
			fields: fields{
				scheme: s,
				client: fake.NewFakeClient([]runtime.Object{
					kanaryv1alpha1test.NewKanaryStatefulset(name, namespace, "", defaultReplicas, &kanaryv1alpha1test.NewKanaryStatefulsetOptions{
						Status: &kanaryv1alpha1.KanaryStatefulsetStatus{
							Conditions: []kanaryv1alpha1.KanaryStatefulsetCondition{
								kanaryv1alpha1.KanaryStatefulsetCondition{
									Status: corev1.ConditionTrue,
									Type:   kanaryv1alpha1.ScheduledKanaryStatefulsetConditionType,
								},
							},
						},
					}),
					utilstest.NewDeployment(name, namespace, defaultReplicas, nil),
				}...),
			},
			want: reconcile.Result{
				Requeue: true,
			},
			wantFunc: func(r *ReconcileKanaryStatefulset) error {
				deployment := &appsv1beta1.Deployment{}
				err := r.client.Get(context.TODO(), types.NamespacedName{Name: name + "-kanary-" + name, Namespace: namespace}, deployment)
				if err != nil && errors.IsNotFound(err) {
					return fmt.Errorf("unable to get the created canary deployment, %v", err)
				}
				if err != nil {
					return err
				}
				// check if replicas is equal to 0
				if deployment.Spec.Replicas == nil {
					return fmt.Errorf("replicas should not be nil")
				} else if *deployment.Spec.Replicas != int32(1) {
					return fmt.Errorf("replicas should be equal to 1, current value %d", *deployment.Spec.Replicas)
				}

				return nil
			},
		},
		{
			name: "[INIT] service is not defined",

			request: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      name,
					Namespace: namespace,
				},
			},
			fields: fields{
				scheme: s,
				client: fake.NewFakeClient([]runtime.Object{
					kanaryv1alpha1test.NewKanaryStatefulset(name, namespace, "", defaultReplicas, &kanaryv1alpha1test.NewKanaryStatefulsetOptions{
						Status: &kanaryv1alpha1.KanaryStatefulsetStatus{
							Conditions: []kanaryv1alpha1.KanaryStatefulsetCondition{
								kanaryv1alpha1.KanaryStatefulsetCondition{
									Status: corev1.ConditionTrue,
									Type:   kanaryv1alpha1.ScheduledKanaryStatefulsetConditionType,
								},
							},
						},
					}),
					utilstest.NewDeployment(name, namespace, defaultReplicas, nil),
					utilstest.NewDeployment(name+"-kanary-"+name, namespace, 1, nil),
				}...),
			},
			want: reconcile.Result{
				Requeue: true,
			},
			wantFunc: func(r *ReconcileKanaryStatefulset) error {
				service := &corev1.Service{}
				err := r.client.Get(context.TODO(), types.NamespacedName{Name: name + "-kanary", Namespace: namespace}, service)
				if err != nil && errors.IsNotFound(err) {
					return nil
				}
				return err
			},
		},
		{
			name: "[INIT] service is define but dont exist",

			request: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      name,
					Namespace: namespace,
				},
			},
			fields: fields{
				scheme: s,
				client: fake.NewFakeClient([]runtime.Object{
					kanaryv1alpha1test.NewKanaryStatefulset(name, namespace, serviceName, defaultReplicas, &kanaryv1alpha1test.NewKanaryStatefulsetOptions{Traffic: kanaryServiceTraffic,
						Status: &kanaryv1alpha1.KanaryStatefulsetStatus{
							Conditions: []kanaryv1alpha1.KanaryStatefulsetCondition{
								kanaryv1alpha1.KanaryStatefulsetCondition{
									Status: corev1.ConditionTrue,
									Type:   kanaryv1alpha1.ScheduledKanaryStatefulsetConditionType,
								},
							},
						},
					}),
					utilstest.NewDeployment(name, namespace, defaultReplicas, nil),
					utilstest.NewDeployment(name+"-kanary-"+name, namespace, 1, nil),
				}...),
			},
			want: reconcile.Result{
				Requeue:      true,
				RequeueAfter: time.Duration(1 * time.Second),
			},
			wantErr: true,
			wantFunc: func(r *ReconcileKanaryStatefulset) error {
				service := &corev1.Service{}
				err := r.client.Get(context.TODO(), types.NamespacedName{Name: name + "-kanary", Namespace: namespace}, service)
				if err != nil && errors.IsNotFound(err) {
					return nil
				}
				return err
			},
		},

		{
			name: "[INIT] service is define, test kanary service creation",

			request: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      name,
					Namespace: namespace,
				},
			},
			fields: fields{
				scheme: s,
				client: fake.NewFakeClient([]runtime.Object{
					kanaryv1alpha1test.NewKanaryStatefulset(name, namespace, serviceName, defaultReplicas, &kanaryv1alpha1test.NewKanaryStatefulsetOptions{Traffic: kanaryServiceTraffic,
						Status: &kanaryv1alpha1.KanaryStatefulsetStatus{
							Conditions: []kanaryv1alpha1.KanaryStatefulsetCondition{
								kanaryv1alpha1.KanaryStatefulsetCondition{
									Status: corev1.ConditionTrue,
									Type:   kanaryv1alpha1.ScheduledKanaryStatefulsetConditionType,
								},
							},
						},
					}),
					utilstest.NewDeployment(name, namespace, defaultReplicas, nil),
					utilstest.NewDeployment(name+"-kanary-"+name, namespace, 1, nil),
					utilstest.NewService(serviceName, namespace, nil, nil),
				}...),
			},
			want: reconcile.Result{
				Requeue: true,
			},
			wantFunc: func(r *ReconcileKanaryStatefulset) error {
				service := &corev1.Service{}
				err := r.client.Get(context.TODO(), types.NamespacedName{Name: serviceName + "-kanary-" + name, Namespace: namespace}, service)
				if err != nil && errors.IsNotFound(err) {
					return fmt.Errorf("unable to get the created canary service, %v", err)
				}
				labelFound := false
				for key, val := range service.Spec.Selector {
					if key == kanaryv1alpha1.KanaryStatefulsetActivateLabelKey && val == kanaryv1alpha1.KanaryStatefulsetLabelValueTrue {
						labelFound = true
						break
					}
				}

				if !labelFound {
					return fmt.Errorf("unable to found the label key: %s in the service.Spec.Selector map", kanaryv1alpha1.KanaryStatefulsetActivateLabelKey)
				}
				return err
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &ReconcileKanaryStatefulset{
				client: tt.fields.client,
				scheme: tt.fields.scheme,
			}
			got, err := r.Reconcile(tt.request)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReconcileKanaryStatefulset.Reconcile() wantErr error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ReconcileKanaryStatefulset.Reconcile() = %v, want %v", got, tt.want)
			}
			if tt.wantFunc != nil {
				if err := tt.wantFunc(r); err != nil {
					t.Errorf("ReconcileKanaryStatefulset.Reconcile() not properly validated, %v", err)
				}
			}
		})
	}
}
