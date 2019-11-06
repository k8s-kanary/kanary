package strategies

import (
	"fmt"
	"os"
	"strings"

	"github.com/go-logr/logr"

	kruisev1alpha1 "github.com/openkruise/kruise/pkg/apis/apps/v1alpha1"
	appsv1beta1 "k8s.io/api/apps/v1beta1"
	corev1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	kanaryv1alpha1 "github.com/k8s-kanary/kanary/pkg/apis/kanary/v1alpha1"
	"github.com/k8s-kanary/kanary/pkg/config"
	"github.com/k8s-kanary/kanary/pkg/controller/kanarystatefulset/strategies/validation"
	"github.com/k8s-kanary/kanary/pkg/controller/kanarystatefulset/utils"
)

// Interface represent the strategy interface
type Interface interface {
	Apply(kclient client.Client, reqLogger logr.Logger, kd *kanaryv1alpha1.KanaryStatefulset, dep, canarydep *appsv1beta1.Deployment, statefulst *kruisev1alpha1.StatefulSet) (result reconcile.Result, err error)
}

// NewStrategy return new instance of the strategy
func NewStrategy(spec *kanaryv1alpha1.KanaryStatefulsetSpec) (Interface, error) {
	var validationsImpls []validation.Interface
	for _, v := range spec.Validations.Items {
		if v.Manual != nil {
			validationsImpls = append(validationsImpls, validation.NewManual(&spec.Validations, &v))
		} else if v.LabelWatch != nil {
			validationsImpls = append(validationsImpls, validation.NewLabelWatch(&spec.Validations, &v))
		} else if v.PromQL != nil {
			validationsImpls = append(validationsImpls, validation.NewPromql(&spec.Validations, &v))
		}
	}
	return &strategy{
		validations:         validationsImpls,
		subResourceDisabled: os.Getenv(config.KanaryStatusSubresourceDisabledEnvVar) == "1",
	}, nil
}

type strategy struct {
	validations         []validation.Interface
	subResourceDisabled bool
}

func (s *strategy) Apply(kclient client.Client, reqLogger logr.Logger, kd *kanaryv1alpha1.KanaryStatefulset, dep, canarydep *appsv1beta1.Deployment, statefulst *kruisev1alpha1.StatefulSet) (result reconcile.Result, err error) {
	var newStatus *kanaryv1alpha1.KanaryStatefulsetStatus
	newStatus, result, err = s.process(kclient, reqLogger, kd, dep, canarydep, statefulst)
	utils.UpdateKanaryStatefulsetStatusConditionsFailure(newStatus, metav1.Now(), err)
	return utils.UpdateKanaryStatefulsetStatus(kclient, s.subResourceDisabled, reqLogger, kd, newStatus, result, err) //Try with plain resource
}

func (s *strategy) process(kclient client.Client, reqLogger logr.Logger, kd *kanaryv1alpha1.KanaryStatefulset, dep, canarydep *appsv1beta1.Deployment, sts *kruisev1alpha1.StatefulSet) (*kanaryv1alpha1.KanaryStatefulsetStatus, reconcile.Result, error) {

	//before going to validation step, let's check that initial delay period is completed
	if reaminingDelay, done := validation.IsInitialDelayDone(kd); !done {
		reqLogger.Info("Check Validation", "requeue-initial-delay", reaminingDelay)
		return &kd.Status, reconcile.Result{RequeueAfter: reaminingDelay}, nil
	}

	//In ase we are still running let's run the validation
	if !utils.IsKanaryStatefulsetValidationCompleted(&kd.Status) {
		reqLogger.Info("Check Validation")

		if !utils.IsKanaryStatefulsetValidationRunning(&kd.Status) {
			status := kd.Status.DeepCopy()
			utils.UpdateKanaryStatefulsetStatusCondition(status, metav1.Now(), kanaryv1alpha1.RunningKanaryStatefulsetConditionType, corev1.ConditionTrue, "Validation Started", false)
			reqLogger.Info("Validation Started")
			return status, reconcile.Result{Requeue: true}, nil
		}

		validationDeadlineDone := validation.IsDeadlinePeriodDone(kd)

		//Run validation for all strategies
		var results []*validation.Result
		var errs []error
		for _, validationItem := range s.validations {
			var result *validation.Result
			result, err := validationItem.Validation(kclient, reqLogger, kd, dep, canarydep, sts)
			if err != nil {
				errs = append(errs, err)
			}
			results = append(results, result)
		}
		if len(errs) > 0 {
			return &kd.Status, reconcile.Result{Requeue: true}, utilerrors.NewAggregate(errs)
		}

		var forceSucceededNow bool
		var failMessages string
		failMessages, forceSucceededNow = computeStatus(results)
		failed := failMessages != ""

		// If any strategy fails, the kanary should fail
		if failed {
			reqLogger.Info("Check Validation failed")
			status := kd.Status.DeepCopy()
			utils.UpdateKanaryStatefulsetStatusCondition(status, metav1.Now(), kanaryv1alpha1.FailedKanaryStatefulsetConditionType, corev1.ConditionTrue, fmt.Sprintf("KanaryStatefulset failed, %s", failMessages), false)
			utils.UpdateKanaryStatefulsetStatusCondition(status, metav1.Now(), kanaryv1alpha1.RunningKanaryStatefulsetConditionType, corev1.ConditionFalse, "Validation ended with failure detected", false)
			reqLogger.Info("Check Validation", "in failed", failMessages, "updated status", fmt.Sprintf("%#v", status))
			return status, reconcile.Result{Requeue: true}, nil
		}

		// So there is no failure, does someone force for an early Success ?
		if forceSucceededNow {
			reqLogger.Info("Check Validation success")
			status := kd.Status.DeepCopy()
			utils.UpdateKanaryStatefulsetStatusCondition(status, metav1.Now(), kanaryv1alpha1.SucceededKanaryStatefulsetConditionType, corev1.ConditionTrue, "Forced Success", false)
			utils.UpdateKanaryStatefulsetStatusCondition(status, metav1.Now(), kanaryv1alpha1.RunningKanaryStatefulsetConditionType, corev1.ConditionFalse, "Validation ended with success forced", false)
			return status, reconcile.Result{Requeue: true}, nil
		}

		// No failure, so if we have not reached the validation deadline, let's requeue for next validation
		if !validationDeadlineDone && !failed {
			reqLogger.Info("Check Validation others")
			d := validation.GetNextValidationCheckDuration(kd)
			reqLogger.Info("Check Validation", "Periodic-Requeue", d)
			return &kd.Status, reconcile.Result{RequeueAfter: d}, nil
		}

		// Validation completed and everything is ok while we have reached the end of the validation period...

		//Particular case of the manual strategy with None as StatusAfterDeadline
		if validation.IsStatusAfterDeadlineNone(kd) {
			// No automation, no requeue, wait for manual input
			return &kd.Status, reconcile.Result{}, nil
		}

		//Looks like it is a success for the kanary!
		status := kd.Status.DeepCopy()
		utils.UpdateKanaryStatefulsetStatusCondition(status, metav1.Now(), kanaryv1alpha1.SucceededKanaryStatefulsetConditionType, corev1.ConditionTrue, "Validation ended with success", false)
		utils.UpdateKanaryStatefulsetStatusCondition(status, metav1.Now(), kanaryv1alpha1.RunningKanaryStatefulsetConditionType, corev1.ConditionFalse, "Validation ended with success", false)
		return status, reconcile.Result{Requeue: true}, nil
	}

	//In case of succeeded kanary, we may need to update the deployment
	if utils.IsKanaryStatefulsetSucceeded(&kd.Status) {
		reqLogger.Info("check kanary success")
		if kd.Spec.Validations.NoUpdate {
			return &kd.Status, reconcile.Result{}, nil // nothing else to do... the kanary succeeded, and we are in dry-run mode
		}
		status := kd.Status.DeepCopy()
		// utils.UpdateKanaryStatefulsetStatusCondition(status, metav1.Now(), kanaryv1alpha1.DeploymentUpdatedKanaryStatefulsetConditionType, corev1.ConditionTrue, "Deployment updated successfully", false)
		return status, reconcile.Result{Requeue: true}, nil
	}

	//In case of succeeded kanary, we may need to update the deployment
	if utils.IsKanaryStatefulsetFailed(&kd.Status) {
		if kd.Spec.Validations.NoUpdate {
			return &kd.Status, reconcile.Result{}, nil
		}
		// TODO: 回滚
		reqLogger.Info("check kanary failed, todo go back")
	}

	return &kd.Status, reconcile.Result{}, nil
}

const (
	unknownFailureReason = "unknown failure reason"
)

func computeStatus(results []*validation.Result) (failMessages string, forceSuccessNow bool) {
	if len(results) == 0 {
		return "", forceSuccessNow
	}
	forceSuccessNow = true

	comments := []string{}
	for _, result := range results {

		if !result.ForceSuccessNow {
			forceSuccessNow = false
		}
		if result.IsFailed {
			if result.Comment != "" {
				comments = append(comments, result.Comment)
			} else {
				comments = append(comments, unknownFailureReason)
			}
		}
	}
	if len(comments) > 0 {
		failMessages = strings.Join(comments, ",")
	}
	return failMessages, forceSuccessNow
}

func needReturn(result *reconcile.Result) bool {
	if result.Requeue || int64(result.RequeueAfter) > int64(0) {
		return true
	}
	return false
}
