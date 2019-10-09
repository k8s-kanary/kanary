package utils

import (
	"context"
	"fmt"

	appsv1beta1 "k8s.io/api/apps/v1beta1"
	corev1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/k8s-kanary/kanary/pkg/apis"
	kanaryv1alpha1 "github.com/k8s-kanary/kanary/pkg/apis/kanary/v1alpha1"
	"github.com/k8s-kanary/kanary/pkg/controller/kanarystatefulset/utils/comparison"
)

//PrepareSchemeForOwnerRef return the scheme required to write the kanary ownerreference
func PrepareSchemeForOwnerRef() *runtime.Scheme {
	scheme := runtime.NewScheme()
	if err := apis.AddToScheme(scheme); err != nil {
		panic(err.Error())
	}
	return scheme
}

// NewCanaryServiceForKanaryStatefulset returns a Service object
func NewCanaryServiceForKanaryStatefulset(kd *kanaryv1alpha1.KanaryStatefulset, service *corev1.Service, overwriteLabel bool, scheme *runtime.Scheme, setOwnerRef bool) (*corev1.Service, error) {
	kanaryServiceName := GetCanaryServiceName(kd)

	labelSelector := map[string]string{}
	labelSelector[kanaryv1alpha1.KanaryStatefulsetKanaryNameLabelKey] = kd.Name
	labelSelector[kanaryv1alpha1.KanaryStatefulsetActivateLabelKey] = kanaryv1alpha1.KanaryStatefulsetLabelValueTrue

	newService := service.DeepCopy()
	newService.ObjectMeta = metav1.ObjectMeta{
		Name:      kanaryServiceName,
		Namespace: kd.Namespace,
	}
	newService.Spec.Selector = labelSelector
	if newService.Spec.Type == corev1.ServiceTypeNodePort || newService.Spec.Type == corev1.ServiceTypeLoadBalancer {
		// this is to remove Port collision
		if newService.Spec.Type == corev1.ServiceTypeNodePort {
			for i := range newService.Spec.Ports {
				newService.Spec.Ports[i].NodePort = 0
			}
		}
		if newService.Spec.Type == corev1.ServiceTypeLoadBalancer {
			newService.Spec.LoadBalancerSourceRanges = nil
		}

		newService.Spec.Type = corev1.ServiceTypeClusterIP
	}
	newService.Spec.ClusterIP = ""
	newService.Status = corev1.ServiceStatus{}

	if setOwnerRef {
		// Set KanaryStatefulset instance as the owner and controller
		if err := controllerutil.SetControllerReference(kd, newService, scheme); err != nil {
			return nil, err
		}
	}
	return newService, nil
}

// GetCanaryServiceName returns the canary service name depending of the spec
func GetCanaryServiceName(kd *kanaryv1alpha1.KanaryStatefulset) string {
	kanaryServiceName := kd.Spec.Traffic.KanaryService
	if kanaryServiceName == "" {
		kanaryServiceName = fmt.Sprintf("%s-kanary-%s", kd.Spec.ServiceName, kd.Name)
	}
	return kanaryServiceName
}

// NewDeploymentFromKanaryStatefulsetTemplate returns a Deployment object
func NewDeploymentFromKanaryStatefulsetTemplate(kdold *kanaryv1alpha1.KanaryStatefulset, scheme *runtime.Scheme, setOwnerRef bool) (*appsv1beta1.Deployment, error) {
	kd := kdold.DeepCopy()
	ls := GetLabelsForKanaryStatefulsetd(kd.Name)

	dep := &appsv1beta1.Deployment{
		TypeMeta:   kd.Spec.Template.TypeMeta,
		ObjectMeta: kd.Spec.Template.ObjectMeta,
		Spec:       kd.Spec.Template.Spec,
	}

	if dep.Labels == nil {
		dep.Labels = map[string]string{}
	}

	for key, val := range ls {
		dep.Labels[key] = val
	}

	dep.Name = GetDeploymentName(kd)
	if dep.Namespace == "" {
		dep.Namespace = kd.Namespace
	}

	if _, err := comparison.SetMD5DeploymentSpecAnnotation(kd, dep); err != nil {
		return nil, fmt.Errorf("unable to set the md5 annotation, %v", err)
	}

	if setOwnerRef {
		// Set KanaryStatefulset instance as the owner and controller
		if err := controllerutil.SetControllerReference(kd, dep, scheme); err != nil {
			return dep, err
		}
	}
	return dep, nil
}

// NewCanaryDeploymentFromKanaryStatefulsetTemplate returns a Deployment object
func NewCanaryDeploymentFromKanaryStatefulsetTemplate(kclient client.Client, kd *kanaryv1alpha1.KanaryStatefulset, scheme *runtime.Scheme, setOwnerRef bool) (*appsv1beta1.Deployment, error) {
	dep, err := NewDeploymentFromKanaryStatefulsetTemplate(kd, scheme, true)
	if err != nil {
		return nil, err
	}
	dep.Name = GetCanaryDeploymentName(kd)
	// Overwrite the Pods labels and the Deployment spec selector
	dep.Spec.Template.Labels = map[string]string{
		kanaryv1alpha1.KanaryStatefulsetKanaryNameLabelKey: kd.Name,
		kanaryv1alpha1.KanaryStatefulsetActivateLabelKey:   kanaryv1alpha1.KanaryStatefulsetLabelValueTrue,
	}
	dep.Spec.Selector.MatchLabels = dep.Spec.Template.Labels

	//Here add the labels that are not part of the service selector
	service := &corev1.Service{}
	err = kclient.Get(context.TODO(), types.NamespacedName{Name: kd.Spec.ServiceName, Namespace: kd.Namespace}, service)
	serviceSelector := service.Spec.Selector
	if err == nil {
		for k, v := range kd.Spec.Template.Spec.Template.ObjectMeta.Labels {
			if _, ok := serviceSelector[k]; ok {
				continue // don't add this label that is used by service discovery. The traffic strategy will add it if needed
			}
			dep.Spec.Template.Labels[k] = v //typically add labels like "version" that are used for pod management but not for service discovery
		}
	}

	dep.Spec.Replicas = GetCanaryReplicasValue(kd)

	return dep, nil
}

// UpdateDeploymentWithKanaryStatefulsetTemplate returns a Deployment object updated
func UpdateDeploymentWithKanaryStatefulsetTemplate(kd *kanaryv1alpha1.KanaryStatefulset, oldDep *appsv1beta1.Deployment) (*appsv1beta1.Deployment, error) {
	newDep := oldDep.DeepCopy()
	{
		newDep.Labels = kd.Spec.Template.Labels
		newDep.Annotations = kd.Spec.Template.Annotations
		newDep.Spec = kd.Spec.Template.Spec
	}

	if _, err := comparison.SetMD5DeploymentSpecAnnotation(kd, newDep); err != nil {
		return nil, fmt.Errorf("unable to set the md5 annotation, %v", err)
	}

	return newDep, nil
}

// GetDeploymentName returns the Deployment name from the KanaryStatefulset instance
func GetDeploymentName(kd *kanaryv1alpha1.KanaryStatefulset) string {
	name := kd.Spec.Template.ObjectMeta.Name
	if kd.Spec.DeploymentName != "" {
		name = kd.Spec.DeploymentName
	} else if name == "" {
		name = kd.Name
	}
	return name
}

// GetCanaryDeploymentName returns the Canary Deployment name from the KanaryStatefulset instance
func GetCanaryDeploymentName(kd *kanaryv1alpha1.KanaryStatefulset) string {
	return fmt.Sprintf("%s-kanary-%s", GetDeploymentName(kd), kd.Name)
}


// GetLabelsForKanaryStatefulsetd return labels belonging to the given KanaryStatefulset CR name.
func GetLabelsForKanaryStatefulsetd(name string) map[string]string {
	return map[string]string{
		kanaryv1alpha1.KanaryStatefulsetIsKanaryLabelKey:   kanaryv1alpha1.KanaryStatefulsetLabelValueTrue,
		kanaryv1alpha1.KanaryStatefulsetKanaryNameLabelKey: name,
	}
}

// GetLabelsForKanaryPod return labels of a canary pod associated to a kanarystatefulset.
func GetLabelsForKanaryPod(kdname string) map[string]string {
	return map[string]string{
		kanaryv1alpha1.KanaryStatefulsetKanaryNameLabelKey: kdname,
		kanaryv1alpha1.KanaryStatefulsetActivateLabelKey:   kanaryv1alpha1.KanaryStatefulsetLabelValueTrue,
	}
}

// GetCanaryReplicasValue returns the replicas value of the Canary Deployment
func GetCanaryReplicasValue(kd *kanaryv1alpha1.KanaryStatefulset) *int32 {
	var value *int32
	if kd.Spec.Scale.Static != nil {
		value = kd.Spec.Scale.Static.Replicas
	}
	return value
}
