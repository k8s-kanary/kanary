package traffic

import (
	"github.com/go-logr/logr"

	appsv1beta1 "k8s.io/api/apps/v1beta1"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	
	kruisev1alpha1 "github.com/openkruise/kruise/pkg/apis/apps/v1alpha1"
	kanaryv1alpha1 "github.com/k8s-kanary/kanary/pkg/apis/kanary/v1alpha1"
)

// NewMirror returns new traffic.Live instance
func NewMirror(s *kanaryv1alpha1.KanaryStatefulsetSpecTraffic) Interface {
	return &mirrorImpl{
		conf: s.Mirror,
	}
}

type mirrorImpl struct {
	conf *kanaryv1alpha1.KanaryStatefulsetSpecTrafficMirror
}

func (s *mirrorImpl) Traffic(kclient client.Client, reqLogger logr.Logger, kd *kanaryv1alpha1.KanaryStatefulset, canaryDep *appsv1beta1.Deployment, sts *kruisev1alpha1.StatefulSet) (status *kanaryv1alpha1.KanaryStatefulsetStatus, result reconcile.Result, err error) {
	return
}

func (s *mirrorImpl) Cleanup(kclient client.Client, reqLogger logr.Logger, kd *kanaryv1alpha1.KanaryStatefulset, canaryDep *appsv1beta1.Deployment, sts *kruisev1alpha1.StatefulSet) (status *kanaryv1alpha1.KanaryStatefulsetStatus, result reconcile.Result, err error) {
	return
}
