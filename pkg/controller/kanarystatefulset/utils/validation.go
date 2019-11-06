package utils

import (
	"fmt"

	"github.com/k8s-kanary/kanary/pkg/apis/kanary/v1alpha1"
)

// ValidateKanaryStatefulset used to validate a KanaryStatefulset
// return a list of errors in case of unvalid fields.
func ValidateKanaryStatefulset(kd *v1alpha1.KanaryStatefulset) []error {
	var errs []error
	errs = append(errs, validateKanaryStatefulsetSpecValidationList(&kd.Spec.Validations)...)
	return errs
}


func validateKanaryStatefulsetSpecValidationList(list *v1alpha1.KanaryStatefulsetSpecValidationList) []error {
	var errs []error
	if len(list.Items) == 0 {
		return []error{fmt.Errorf("validation list is not set")}
	}
	for _, v := range list.Items {
		errs = append(errs, validateKanaryStatefulsetSpecValidation(&v)...)
	}
	return errs
}

func validateKanaryStatefulsetSpecValidation(v *v1alpha1.KanaryStatefulsetSpecValidation) []error {
	var errs []error
	if v.Manual == nil && v.LabelWatch == nil && v.PromQL == nil {
		errs = append(errs, fmt.Errorf("spec.validation not defined: %v", v))
	}

	return errs
}
