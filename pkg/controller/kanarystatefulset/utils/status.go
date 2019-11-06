package utils

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-logr/logr"

	corev1 "k8s.io/api/core/v1"

	apiequality "k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/k8s-kanary/kanary/pkg/apis/kanary/v1alpha1"
	kanaryv1alpha1 "github.com/k8s-kanary/kanary/pkg/apis/kanary/v1alpha1"
)

// UpdateKanaryStatefulsetStatus used to update the KanaryStatefulset.Status if it has changed.
func UpdateKanaryStatefulsetStatus(kclient client.Client, subResourceDisabled bool, reqLogger logr.Logger, kd *kanaryv1alpha1.KanaryStatefulset, newStatus *kanaryv1alpha1.KanaryStatefulsetStatus, result reconcile.Result, err error) (reconcile.Result, error) {

	var kclientStatus client.StatusWriter = kclient
	if !subResourceDisabled { //Updating StatusSubresource may depends on Kubernetes version! https://book.kubebuilder.io/basics/status_subresource.html
		kclientStatus = kclient.Status()
	}

	//No need to go further if the same pointer is given
	if &kd.Status == newStatus {
		return result, err
	}

	updateStatusReport(kd, newStatus)
	if !apiequality.Semantic.DeepEqual(&kd.Status, newStatus) {
		updatedKd := kd.DeepCopy()
		updatedKd.Status = *newStatus
		err2 := kclientStatus.Update(context.TODO(), updatedKd)
		if err2 != nil {
			reqLogger.Error(err2, "failed to update KanaryStatefulset status", "KanaryStatefulset.Namespace", updatedKd.Namespace, "KanaryStatefulset.Name", updatedKd.Name)
			return reconcile.Result{}, err2
		}
	}
	return result, err
}

// UpdateKanaryStatefulsetStatusConditionsFailure used to update the failre StatusConditions
func UpdateKanaryStatefulsetStatusConditionsFailure(status *kanaryv1alpha1.KanaryStatefulsetStatus, now metav1.Time, err error) {
	if err != nil {
		UpdateKanaryStatefulsetStatusCondition(status, now, kanaryv1alpha1.ErroredKanaryStatefulsetConditionType, corev1.ConditionTrue, fmt.Sprintf("%v", err), false)
	} else {
		UpdateKanaryStatefulsetStatusCondition(status, now, kanaryv1alpha1.ErroredKanaryStatefulsetConditionType, corev1.ConditionFalse, "", false)
	}
}

// UpdateKanaryStatefulsetStatusCondition used to update a specific KanaryStatefulsetConditionType
func UpdateKanaryStatefulsetStatusCondition(status *kanaryv1alpha1.KanaryStatefulsetStatus, now metav1.Time, t kanaryv1alpha1.KanaryStatefulsetConditionType, conditionStatus corev1.ConditionStatus, desc string, writeFalseIfNotExist bool) {
	idConditionComplete := getIndexForConditionType(status, t)
	if idConditionComplete >= 0 {
		if status.Conditions[idConditionComplete].Status != conditionStatus {
			status.Conditions[idConditionComplete].LastTransitionTime = now
			status.Conditions[idConditionComplete].Status = conditionStatus
		}
		status.Conditions[idConditionComplete].LastUpdateTime = now
		status.Conditions[idConditionComplete].Message = desc
	} else if conditionStatus == corev1.ConditionTrue || writeFalseIfNotExist {
		// Only add if the condition is True
		status.Conditions = append(status.Conditions, NewKanaryStatefulsetStatusCondition(t, conditionStatus, now, "", desc))
	}
}

// NewKanaryStatefulsetStatusCondition returns new KanaryStatefulsetCondition instance
func NewKanaryStatefulsetStatusCondition(conditionType kanaryv1alpha1.KanaryStatefulsetConditionType, conditionStatus corev1.ConditionStatus, now metav1.Time, reason, message string) kanaryv1alpha1.KanaryStatefulsetCondition {
	return kanaryv1alpha1.KanaryStatefulsetCondition{
		Type:               conditionType,
		Status:             conditionStatus,
		LastUpdateTime:     now,
		LastTransitionTime: now,
		Reason:             reason,
		Message:            message,
	}
}

// IsKanaryStatefulsetErrored returns true if the KanaryStatefulset has failed, else returns false
func IsKanaryStatefulsetErrored(status *kanaryv1alpha1.KanaryStatefulsetStatus) bool {
	if status == nil {
		return false
	}
	id := getIndexForConditionType(status, kanaryv1alpha1.ErroredKanaryStatefulsetConditionType)
	if id >= 0 && status.Conditions[id].Status == corev1.ConditionTrue {
		return true
	}
	return false
}

// IsKanaryStatefulsetFailed returns true if the KanaryStatefulset has failed, else returns false
func IsKanaryStatefulsetFailed(status *kanaryv1alpha1.KanaryStatefulsetStatus) bool {
	if status == nil {
		return false
	}
	id := getIndexForConditionType(status, kanaryv1alpha1.FailedKanaryStatefulsetConditionType)
	if id >= 0 && status.Conditions[id].Status == corev1.ConditionTrue {
		return true
	}
	return false
}

// IsKanaryStatefulsetSucceeded returns true if the KanaryStatefulset has succeeded, else return false
func IsKanaryStatefulsetSucceeded(status *kanaryv1alpha1.KanaryStatefulsetStatus) bool {
	if status == nil {
		return false
	}
	id := getIndexForConditionType(status, kanaryv1alpha1.SucceededKanaryStatefulsetConditionType)
	if id >= 0 && status.Conditions[id].Status == corev1.ConditionTrue {
		return true
	}
	return false
}

// IsKanaryStatefulsetScheduled returns true if the KanaryStatefulset is scheduled, else return false
func IsKanaryStatefulsetScheduled(status *kanaryv1alpha1.KanaryStatefulsetStatus) bool {
	if status == nil {
		return false
	}
	id := getIndexForConditionType(status, kanaryv1alpha1.ScheduledKanaryStatefulsetConditionType)
	if id >= 0 && status.Conditions[id].Status == corev1.ConditionTrue {
		return true
	}
	return false
}

// IsKanaryStatefulsetDeploymentUpdated returns true if the KanaryStatefulset has lead to newstatefulset update
// return all true
func IsKanaryStatefulsetUpdated(status *kanaryv1alpha1.KanaryStatefulsetStatus) bool {
	return true
}

// IsKanaryStatefulsetValidationRunning returns true if the KanaryStatefulset is runnning
func IsKanaryStatefulsetValidationRunning(status *kanaryv1alpha1.KanaryStatefulsetStatus) bool {
	if status == nil {
		return false
	}
	id := getIndexForConditionType(status, kanaryv1alpha1.RunningKanaryStatefulsetConditionType)
	if id >= 0 && status.Conditions[id].Status == corev1.ConditionTrue {
		return true
	}
	return false
}

// IsKanaryStatefulsetValidationCompleted returns true if the KanaryStatefulset is runnning
func IsKanaryStatefulsetValidationCompleted(status *kanaryv1alpha1.KanaryStatefulsetStatus) bool {
	return IsKanaryStatefulsetFailed(status) || IsKanaryStatefulsetSucceeded(status) || IsKanaryStatefulsetUpdated(status)
}

func getIndexForConditionType(status *kanaryv1alpha1.KanaryStatefulsetStatus, t kanaryv1alpha1.KanaryStatefulsetConditionType) int {
	idCondition := -1
	if status == nil {
		return idCondition
	}
	for i, condition := range status.Conditions {
		if condition.Type == t {
			idCondition = i
			break
		}
	}
	return idCondition
}

func getReportStatus(status *kanaryv1alpha1.KanaryStatefulsetStatus) string {

	// Order matters compare to the lifecycle of the kanary during validation

	if IsKanaryStatefulsetFailed(status) {
		return string(v1alpha1.FailedKanaryStatefulsetConditionType)
	}

	if IsKanaryStatefulsetUpdated(status) {
		return "kanary statefulset updated"
	}

	if IsKanaryStatefulsetSucceeded(status) {
		return string(v1alpha1.SucceededKanaryStatefulsetConditionType)
	}

	if IsKanaryStatefulsetValidationRunning(status) {
		return string(v1alpha1.RunningKanaryStatefulsetConditionType)
	}

	if IsKanaryStatefulsetScheduled(status) {
		return string(v1alpha1.ScheduledKanaryStatefulsetConditionType)
	}

	if IsKanaryStatefulsetErrored(status) {
		return string(v1alpha1.ErroredKanaryStatefulsetConditionType)
	}

	return "-"
}

func getValidation(kd *kanaryv1alpha1.KanaryStatefulset) string {
	var list []string
	for _, v := range kd.Spec.Validations.Items {
		if v.LabelWatch != nil {
			list = append(list, "labelWatch")
		}
		if v.PromQL != nil {
			list = append(list, "promQL")
		}
		if v.Manual != nil {
			list = append(list, "manual")
		}
	}
	if len(list) == 0 {
		return "unknow"
	}
	return strings.Join(list, ",")
}


func updateStatusReport(kd *kanaryv1alpha1.KanaryStatefulset, status *kanaryv1alpha1.KanaryStatefulsetStatus) {
	status.Report = kanaryv1alpha1.KanaryStatefulsetStatusReport{
		Status:     getReportStatus(status),
		Validation: getValidation(kd),
	}
}
