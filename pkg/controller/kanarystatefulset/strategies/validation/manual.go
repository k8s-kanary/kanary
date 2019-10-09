package validation

import (
	"github.com/go-logr/logr"

	appsv1beta1 "k8s.io/api/apps/v1beta1"

	"sigs.k8s.io/controller-runtime/pkg/client"
	kruisev1alpha1 "github.com/openkruise/kruise/pkg/apis/apps/v1alpha1"
	kanaryv1alpha1 "github.com/k8s-kanary/kanary/pkg/apis/kanary/v1alpha1"
)

// NewManual returns new validation.Manual instance
func NewManual(list *kanaryv1alpha1.KanaryStatefulsetSpecValidationList, s *kanaryv1alpha1.KanaryStatefulsetSpecValidation) Interface {
	return &manualImpl{
		deadlineStatus:         s.Manual.StatusAfterDealine,
		validationManualStatus: s.Manual.Status,
		dryRun:                 list.NoUpdate,
	}
}

type manualImpl struct {
	deadlineStatus         kanaryv1alpha1.KanaryStatefulsetSpecValidationManualDeadineStatus
	validationManualStatus kanaryv1alpha1.KanaryStatefulsetSpecValidationManualStatus
	dryRun                 bool
}

func (m *manualImpl) Validation(kclient client.Client, reqLogger logr.Logger, kd *kanaryv1alpha1.KanaryStatefulset, dep, canaryDep *appsv1beta1.Deployment, sts *kruisev1alpha1.StatefulSet) (*Result, error) {
	var err error
	result := &Result{}

	if m.validationManualStatus == kanaryv1alpha1.ValidKanaryStatefulsetSpecValidationManualStatus {
		result.ForceSuccessNow = true
	}

	deadlineReached := IsDeadlinePeriodDone(kd)

	if m.validationManualStatus == kanaryv1alpha1.ValidKanaryStatefulsetSpecValidationManualStatus {
	} else if m.validationManualStatus == kanaryv1alpha1.InvalidKanaryStatefulsetSpecValidationManualStatus {
		result.IsFailed = true
		result.Comment = "manual.status=invalid"
	} else if deadlineReached && m.deadlineStatus == kanaryv1alpha1.InvalidKanaryStatefulsetSpecValidationManualDeadineStatus {
		result.IsFailed = true
		result.Comment = "deadline activated with 'invalid' status"
	} else if deadlineReached && m.deadlineStatus == kanaryv1alpha1.ValidKanaryStatefulsetSpecValidationManualDeadineStatus {
		result.Comment = "deadline activated with 'valid' status"
	}

	return result, err
}

//IsStatusAfterDeadlineNone check if there is a Manual Strategy that prevent automation with a None Status.
func IsStatusAfterDeadlineNone(kd *kanaryv1alpha1.KanaryStatefulset) bool {
	for _, v := range kd.Spec.Validations.Items {
		if v.Manual != nil {
			if v.Manual.StatusAfterDealine == kanaryv1alpha1.NoneKanaryStatefulsetSpecValidationManualDeadineStatus {
				return true
			}
		}
	}
	return false
}
