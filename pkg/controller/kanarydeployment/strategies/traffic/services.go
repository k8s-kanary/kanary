package traffic

import (
	"context"
	"reflect"
	"time"

	"github.com/go-logr/logr"

	appsv1beta1 "k8s.io/api/apps/v1beta1"
	corev1 "k8s.io/api/core/v1"

	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	
	kruisev1alpha1 "github.com/openkruise/kruise/pkg/apis/apps/v1alpha1"
	kanaryv1alpha1 "github.com/amadeusitgroup/kanary/pkg/apis/kanary/v1alpha1"
	"github.com/amadeusitgroup/kanary/pkg/controller/kanarydeployment/utils"
)

// NewKanaryService returns new traffic.KanaryService instance
func NewKanaryService(s *kanaryv1alpha1.KanaryDeploymentSpecTraffic) Interface {
	return &kanaryServiceImpl{
		conf:   s,
		scheme: utils.PrepareSchemeForOwnerRef(),
	}
}

type kanaryServiceImpl struct {
	conf   *kanaryv1alpha1.KanaryDeploymentSpecTraffic
	scheme *runtime.Scheme
}

func (k *kanaryServiceImpl) Traffic(kclient client.Client, reqLogger logr.Logger, kd *kanaryv1alpha1.KanaryDeployment, canaryDep *appsv1beta1.Deployment, sts *kruisev1alpha1.StatefulSet) (*kanaryv1alpha1.KanaryDeploymentStatus, reconcile.Result, error) {
	// Retrieve and create service if defined
	newStatus, needsRequeue, result, err := k.manageServices(kclient, reqLogger, kd)
	utils.UpdateKanaryDeploymentStatusConditionsFailure(newStatus, metav1.Now(), err)
	if needsRequeue {
		result.Requeue = true
	}
	return newStatus, result, err
}

func (k *kanaryServiceImpl) Cleanup(kclient client.Client, reqLogger logr.Logger, kd *kanaryv1alpha1.KanaryDeployment, canaryDep *appsv1beta1.Deployment, sts *kruisev1alpha1.StatefulSet) (status *kanaryv1alpha1.KanaryDeploymentStatus, result reconcile.Result, err error) {
	var needsReturn bool
	if k.conf.Source == kanaryv1alpha1.MirrorKanaryDeploymentSpecTrafficSource || k.conf.Source == kanaryv1alpha1.NoneKanaryDeploymentSpecTrafficSource {
		needsReturn, result, err = k.clearServices(kclient, reqLogger, kd)
		if needsReturn {
			result.Requeue = true
		}
	}

	if (k.conf.Source == kanaryv1alpha1.ServiceKanaryDeploymentSpecTrafficSource || k.conf.Source == kanaryv1alpha1.BothKanaryDeploymentSpecTrafficSource) &&
		utils.IsKanaryDeploymentFailed(&kd.Status) || utils.IsKanaryDeploymentSucceeded(&kd.Status) {
		// in this case remove the pod from live traffic service.
		service := &corev1.Service{}
		err = kclient.Get(context.TODO(), client.ObjectKey{Name: kd.Spec.ServiceName, Namespace: kd.Namespace}, service)
		if err != nil {
			return &kd.Status, reconcile.Result{Requeue: true}, err
		}

		needsReturn, result, err = k.desactivateService(kclient, reqLogger, kd, service)
		if needsReturn {
			result.Requeue = true
		}
	}

	return &kd.Status, result, err
}

// NeedOverwriteSelector used to know if the Deployment.Spec.Selector needs to be overwrited for the kanary deployment
func NeedOverwriteSelector(kd *kanaryv1alpha1.KanaryDeployment) bool {
	// if we dont want that
	switch kd.Spec.Traffic.Source {
	case kanaryv1alpha1.ServiceKanaryDeploymentSpecTrafficSource, kanaryv1alpha1.BothKanaryDeploymentSpecTrafficSource:
		return true
	default:
		return false
	}
}

func (k *kanaryServiceImpl) manageServices(kclient client.Client, reqLogger logr.Logger, kd *kanaryv1alpha1.KanaryDeployment) (*kanaryv1alpha1.KanaryDeploymentStatus, bool, reconcile.Result, error) {
	status := kd.Status.DeepCopy()
	var service *corev1.Service
	var err error

	if kd.Spec.ServiceName != "" {
		service = &corev1.Service{}
		err = kclient.Get(context.TODO(), types.NamespacedName{Name: kd.Spec.ServiceName, Namespace: kd.Namespace}, service)
		if err != nil && errors.IsNotFound(err) {
			// TODO update status, to say that the service didn't exist
			return status, true, reconcile.Result{Requeue: true, RequeueAfter: time.Second}, err
		} else if err != nil {
			reqLogger.Error(err, "failed to get Deployment")
			return status, true, reconcile.Result{}, err
		}
	}

	if service != nil {
		switch k.conf.Source {
		case kanaryv1alpha1.BothKanaryDeploymentSpecTrafficSource, kanaryv1alpha1.KanaryServiceKanaryDeploymentSpecTrafficSource:
			kanaryService, err2 := utils.NewCanaryServiceForKanaryDeployment(kd, service, NeedOverwriteSelector(kd), k.scheme, true)
			if err2 != nil {
				reqLogger.Error(err, "failed to prepare CanaryService", "Namespace", kanaryService.Namespace, "Service.Name", kanaryService.Name)
				return status, true, reconcile.Result{}, err
			}
			currentKanaryService := &corev1.Service{}
			err = kclient.Get(context.TODO(), types.NamespacedName{Name: kanaryService.Name, Namespace: kanaryService.Namespace}, currentKanaryService)
			if err != nil && errors.IsNotFound(err) {
				// Kanary Service does not exist, let's create it
				err = kclient.Create(context.TODO(), kanaryService)
				if err != nil {
					reqLogger.Error(err, "failed to create new CanaryService", "Namespace", kanaryService.Namespace, "Service.Name", kanaryService.Name)
					return status, true, reconcile.Result{}, err
				}
				utils.UpdateKanaryDeploymentStatusCondition(status, metav1.Now(), kanaryv1alpha1.TrafficKanaryDeploymentConditionType, corev1.ConditionTrue, "Traffic source: "+string(k.conf.Source), false)
				// Service created successfully - return and requeue
				return status, true, reconcile.Result{Requeue: true}, nil
			} else if err != nil {
				reqLogger.Error(err, "failed to get Service")
				return status, true, reconcile.Result{}, err
			} else {
				// Kanary Service exist, let's update it if needed
				compareKanaryServiceSpec := kanaryService.Spec.DeepCopy()
				compareCurrentServiceSpec := currentKanaryService.Spec.DeepCopy()
				{
					// remove potential values updated in service.Spec
					compareCurrentServiceSpec.ClusterIP = ""
					compareCurrentServiceSpec.LoadBalancerIP = ""
				}
				if !apiequality.Semantic.DeepEqual(compareKanaryServiceSpec, compareCurrentServiceSpec) {
					updatedService := currentKanaryService.DeepCopy()
					updatedService.Spec = *compareKanaryServiceSpec
					updatedService.Spec.ClusterIP = currentKanaryService.Spec.ClusterIP
					updatedService.Spec.LoadBalancerIP = currentKanaryService.Spec.LoadBalancerIP
					err = kclient.Update(context.TODO(), updatedService)
					if err != nil {
						reqLogger.Error(err, "unable to update the kanary service")
						return status, true, reconcile.Result{}, err
					}
					utils.UpdateKanaryDeploymentStatusCondition(status, metav1.Now(), kanaryv1alpha1.TrafficKanaryDeploymentConditionType, corev1.ConditionTrue, "Traffic source: "+string(k.conf.Source), false)
					// Service updated successfully - return and requeue
					return status, true, reconcile.Result{Requeue: true}, nil
				}
			}
		}

		if k.conf.Source == kanaryv1alpha1.BothKanaryDeploymentSpecTrafficSource || k.conf.Source == kanaryv1alpha1.ServiceKanaryDeploymentSpecTrafficSource {
			var needsReturn bool
			var result reconcile.Result
			if utils.IsKanaryDeploymentFailed(&kd.Status) {
				// in this case remove the pod from live traffic service.
				service = &corev1.Service{}
				err = kclient.Get(context.TODO(), client.ObjectKey{Name: kd.Spec.ServiceName, Namespace: kd.Namespace}, service)
				if err != nil {
					return status, true, reconcile.Result{Requeue: true}, err
				}

				needsReturn, result, err = k.desactivateService(kclient, reqLogger, kd, service)
				utils.UpdateKanaryDeploymentStatusCondition(status, metav1.Now(), kanaryv1alpha1.TrafficKanaryDeploymentConditionType, corev1.ConditionFalse, "Traffic source: "+string(k.conf.Source), false)
				if needsReturn {
					result.Requeue = true
					return status, needsReturn, result, err
				}
			} else {
				needsReturn, result, err = k.updatePodLabels(kclient, reqLogger, kd, service)
				if needsReturn {
					result.Requeue = true
				}
				utils.UpdateKanaryDeploymentStatusCondition(status, metav1.Now(), kanaryv1alpha1.TrafficKanaryDeploymentConditionType, corev1.ConditionTrue, "Traffic source: "+string(k.conf.Source), false)
			}
		}
	}
	return status, false, reconcile.Result{}, err
}

func (k *kanaryServiceImpl) clearServices(kclient client.Client, reqLogger logr.Logger, kd *kanaryv1alpha1.KanaryDeployment) (needsReturn bool, result reconcile.Result, err error) {
	services := &corev1.ServiceList{}

	selector := labels.Set{
		kanaryv1alpha1.KanaryDeploymentKanaryNameLabelKey: kd.Name,
	}

	listOptions := &client.ListOptions{
		LabelSelector: selector.AsSelector(),
		Namespace:     kd.Namespace,
	}
	// use List instead of Get because the spec.ServiceName can empty after a spec configuration change
	err = kclient.List(context.TODO(), listOptions, services)
	if err != nil {
		reqLogger.Error(err, "failed to list Service")
		return true, reconcile.Result{Requeue: true}, err
	}

	var errs []error
	if len(services.Items) > 0 {
		for _, service := range services.Items {
			if _, ok := service.Spec.Selector[kanaryv1alpha1.KanaryDeploymentKanaryNameLabelKey]; !ok {
				// discard this kanary service
				// TODO: remove this test, when fake client support LabelSelection in ListOptions.
				continue
			}
			err = kclient.Delete(context.TODO(), &service)
			if err != nil {
				reqLogger.Error(err, "unable to delete the kanary service")
				errs = append(errs, err)
			}
			needsReturn = true
			result.Requeue = true
		}
	}
	if errs != nil {
		err = utilerrors.NewAggregate(errs)
	}

	return needsReturn, result, err
}

//Set the labels on the kanary pods to match the service
func (k *kanaryServiceImpl) updatePodLabels(kclient client.Client, reqLogger logr.Logger, kd *kanaryv1alpha1.KanaryDeployment, service *corev1.Service) (needsReturn bool, result reconcile.Result, err error) {
	pods := &corev1.PodList{}
	selector := labels.Set{
		kanaryv1alpha1.KanaryDeploymentKanaryNameLabelKey: kd.Name,
	}

	listOptions := &client.ListOptions{
		LabelSelector: selector.AsSelector(),
		Namespace:     kd.Namespace,
	}
	err = kclient.List(context.TODO(), listOptions, pods)
	if err != nil {
		reqLogger.Error(err, "failed to list Service")
		return true, reconcile.Result{Requeue: true}, err
	}
	var errs []error
	var requeue bool
	for _, pod := range pods.Items {
		updatePod := pod.DeepCopy()
		if updatePod.Labels == nil {
			updatePod.Labels = map[string]string{}
		}
		for key, val := range service.Spec.Selector {
			updatePod.Labels[key] = val
		}
		if reflect.DeepEqual(pod.Labels, updatePod.Labels) {
			// labels already configured properly
			continue
		}
		requeue = true

		err = kclient.Update(context.TODO(), updatePod)
		if err != nil {
			errs = append(errs, err)
		}
	}

	return requeue, reconcile.Result{Requeue: requeue}, utilerrors.NewAggregate(errs)
}

//Remove the labels on the kanary pods so that it does not match the service
func (k *kanaryServiceImpl) desactivateService(kclient client.Client, reqLogger logr.Logger, kd *kanaryv1alpha1.KanaryDeployment, service *corev1.Service) (needsReturn bool, result reconcile.Result, err error) {
	var requeue bool
	// in this case remove the pod from live traffic service.
	pods := &corev1.PodList{}
	selector := labels.Set{
		kanaryv1alpha1.KanaryDeploymentKanaryNameLabelKey: kd.Name,
	}

	listOptions := &client.ListOptions{
		LabelSelector: selector.AsSelector(),
		Namespace:     kd.Namespace,
	}
	err = kclient.List(context.TODO(), listOptions, pods)
	if err != nil {
		reqLogger.Error(err, "failed to list Service")
		return true, reconcile.Result{Requeue: true}, err
	}
	var errs []error
	for _, pod := range pods.Items {
		updatePod := pod.DeepCopy()
		if updatePod.Labels == nil {
			continue
		}
		for key := range service.Spec.Selector {
			delete(updatePod.Labels, key)
		}
		if reflect.DeepEqual(pod.Labels, updatePod.Labels) {
			// labels already configured properly
			continue
		}
		requeue = true
		err = kclient.Update(context.TODO(), updatePod)
		if err != nil {
			errs = append(errs, err)
		}
	}

	return requeue, reconcile.Result{Requeue: requeue}, utilerrors.NewAggregate(errs)
}
