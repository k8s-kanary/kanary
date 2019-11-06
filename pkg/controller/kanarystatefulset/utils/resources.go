package utils

import (
	"fmt"

	"github.com/k8s-kanary/kanary/pkg/apis"
	kanaryv1alpha1 "github.com/k8s-kanary/kanary/pkg/apis/kanary/v1alpha1"
	"github.com/k8s-kanary/kanary/pkg/controller/kanarystatefulset/utils/comparison"
	appsv1beta1 "k8s.io/api/apps/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
)

//PrepareSchemeForOwnerRef return the scheme required to write the kanary ownerreference
func PrepareSchemeForOwnerRef() *runtime.Scheme {
	scheme := runtime.NewScheme()
	if err := apis.AddToScheme(scheme); err != nil {
		panic(err.Error())
	}
	return scheme
}

// UpdateStatefulsetWithKanaryStatefulsetTemplate returns a Deployment object updated
func UpdateStatefulsetWithKanaryStatefulsetTemplate(kd *kanaryv1alpha1.KanaryStatefulset, oldSts *appsv1beta1.StatefulSet) (*appsv1beta1.StatefulSet, error) {
	newSts := oldSts.DeepCopy()
	{
		newSts.Labels = kd.Spec.Template.Labels
		newSts.Annotations = kd.Spec.Template.Annotations
		newSts.Spec = kd.Spec.Template.Spec
	}

	if _, err := comparison.SetMD5StatefulsetSpecAnnotation(kd, newSts); err != nil {
		return nil, fmt.Errorf("unable to set the md5 annotation, %v", err)
	}

	return newSts, nil
}

// GetDeploymentName returns the Deployment name from the KanaryStatefulset instance
func GetStatefulsetName(kd *kanaryv1alpha1.KanaryStatefulset) string {
	name := kd.Spec.Template.ObjectMeta.Name
	if kd.Spec.StatefulSetName != "" {
		name = kd.Spec.StatefulSetName
	} else if name == "" {
		name = kd.Name
	}
	return name
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
