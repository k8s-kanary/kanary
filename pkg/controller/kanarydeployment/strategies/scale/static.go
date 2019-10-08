package scale

import (
	"context"

	"github.com/go-logr/logr"

	appsv1beta1 "k8s.io/api/apps/v1beta1"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	
	kuriseclient "github.com/openkruise/kruise/pkg/client"
	kruisev1alpha1 "github.com/openkruise/kruise/pkg/apis/apps/v1alpha1"
	kanaryv1alpha1 "github.com/k8s-kanary/kanary/pkg/apis/kanary/v1alpha1"
	"github.com/k8s-kanary/kanary/pkg/controller/kanarydeployment/utils"
)

// NewStatic returns new scale.Static instance
func NewStatic(s *kanaryv1alpha1.KanaryDeploymentSpecScaleStatic) Interface {
	replicas := int32(1)
	if s != nil {
		replicas = *s.Replicas
	}

	return &staticImpl{
		replicas: &replicas,
	}
}

type staticImpl struct {
	replicas *int32
}

func (s *staticImpl) Scale(kclient client.Client, reqLogger logr.Logger, kd *kanaryv1alpha1.KanaryDeployment, canaryDep *appsv1beta1.Deployment, sts *kruisev1alpha1.StatefulSet) (*kanaryv1alpha1.KanaryDeploymentStatus, reconcile.Result, error) {
	status := &kd.Status
	// don't update the canary deployment replicas if the KanaryDeployment has failed
	if utils.IsKanaryDeploymentFailed(status) {
		return status, reconcile.Result{}, nil
	}

	// check if the canary deployment replicas is up to date
	var specReplicas, canaryReplicas int32
	
	if canaryDep.Spec.Replicas != nil {
		canaryReplicas = *canaryDep.Spec.Replicas
	}
	if s.replicas != nil {
		specReplicas = *s.replicas
	}
	
	if canaryReplicas != specReplicas {
		replicas := int32(1)
		if s.replicas != nil {
			replicas = specReplicas
		}
		
		reqLogger.Info("scale stateful name: ", kd.Spec.StatefulSetName)
		if len(kd.Spec.StatefulSetName) > 0 {
			reqLogger.Info("stateful set needn't scale", replicas)
			return status, reconcile.Result{}, nil
		}
		
		result, err := updateDeploymentReplicas(kclient, reqLogger, canaryDep, replicas)
		return status, result, err
	}

	return status, reconcile.Result{}, nil
}

func (s *staticImpl) Clear(kclient client.Client, reqLogger logr.Logger, kd *kanaryv1alpha1.KanaryDeployment, canaryDep *appsv1beta1.Deployment, sts *kruisev1alpha1.StatefulSet) (*kanaryv1alpha1.KanaryDeploymentStatus, reconcile.Result, error) {
	status := &kd.Status
	return status, reconcile.Result{}, nil
}

func updateDeploymentReplicas(kclient client.Client, reqLogger logr.Logger, dep *appsv1beta1.Deployment, replicas int32) (reconcile.Result, error) {
	updateDep := dep.DeepCopy()
	updateDep.Spec.Replicas = &replicas
	err := kclient.Update(context.TODO(), updateDep)
	if err != nil {
		reqLogger.Error(err, "failed to update Deployment replicas", "Namespace", updateDep.Namespace, "Deployment", updateDep.Name)
	}
	return reconcile.Result{Requeue: true}, err
}

func updateStatefulSetReplicas(reqLogger logr.Logger, sts *kruisev1alpha1.StatefulSet, replicas int32) (reconcile.Result, error) {
	updateSts := sts.DeepCopy()
	updateSts.Spec.Replicas = &replicas
	
	_, err := kuriseclient.GetGenericClient().KruiseClient.AppsV1alpha1().StatefulSets(sts.Namespace).Update(updateSts)
	if err != nil {
		reqLogger.Error(err, "failed to update Deployment replicas", "Namespace", updateSts.Namespace, "Deployment", updateSts.Name)
	}
	return reconcile.Result{Requeue: true}, err
}
