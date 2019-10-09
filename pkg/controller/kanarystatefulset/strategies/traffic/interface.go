package traffic

import (
	"github.com/go-logr/logr"

	appsv1beta1 "k8s.io/api/apps/v1beta1"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	kanaryv1alpha1 "github.com/k8s-kanary/kanary/pkg/apis/kanary/v1alpha1"
	kruisev1alpha1 "github.com/openkruise/kruise/pkg/apis/apps/v1alpha1"
)

// Interface traffic strategy interface
type Interface interface {
	Traffic(kclient client.Client, reqLogger logr.Logger, kd *kanaryv1alpha1.KanaryStatefulset, canaryDep *appsv1beta1.Deployment, sts *kruisev1alpha1.StatefulSet) (*kanaryv1alpha1.KanaryStatefulsetStatus, reconcile.Result, error)
	Cleanup(kclient client.Client, reqLogger logr.Logger, kd *kanaryv1alpha1.KanaryStatefulset, canaryDep *appsv1beta1.Deployment, sts *kruisev1alpha1.StatefulSet) (*kanaryv1alpha1.KanaryStatefulsetStatus, reconcile.Result, error)
}
