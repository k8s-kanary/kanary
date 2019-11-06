package v1alpha1

import (
	"k8s.io/api/apps/v1beta1"
	"k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func init() {
	SchemeBuilder.Register(&KanaryStatefulset{}, &KanaryStatefulsetList{})
}

// KanaryStatefulset is the Schema for the kanarystatefulsets API
type KanaryStatefulset struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KanaryStatefulsetSpec   `json:"spec,omitempty"`
	Status KanaryStatefulsetStatus `json:"status,omitempty"`
}

// KanaryStatefulsetList contains a list of KanaryStatefulset
type KanaryStatefulsetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KanaryStatefulset `json:"items"`
}

// KanaryStatefulsetSpec defines the desired state of KanaryStatefulset
type KanaryStatefulsetSpec struct {
	StatefulSetName string `json:"statefulSetName,omitempty"`
	
	Template StatefulsetTemplate `json:"template,omitempty"`
	Validations KanaryStatefulsetSpecValidationList `json:"validations,omitempty"`
	Schedule string `json:"schedule,omiempty"`
	
	CanaryReplicas int32  `json:"canaryReplicas,omitempty"` // 一开始设置多少个 pod 验证
	Batches int32     `json:"batches,omitempty"` // 后面的 pod 多少批验证
	Pause bool        `json:"pause,omitempty"`
}

// KanaryStatefulsetSpecValidationList define list of KanaryStatefulsetSpecValidation
type KanaryStatefulsetSpecValidationList struct {
	InitialDelay *metav1.Duration `json:"initialDelay,omitempty"`
	ValidationPeriod *metav1.Duration `json:"validationPeriod,omitempty"`
	MaxIntervalPeriod *metav1.Duration `json:"maxIntervalPeriod,omitempty"`
	NoUpdate bool `json:"noUpdate,omitempty"`
	Items []KanaryStatefulsetSpecValidation `json:"items,omitempty"`
}

// KanaryStatefulsetSpecValidation defines the validation configuration for the canary deployment
type KanaryStatefulsetSpecValidation struct {
	Manual     *KanaryStatefulsetSpecValidationManual     `json:"manual,omitempty"`
	LabelWatch *KanaryStatefulsetSpecValidationLabelWatch `json:"labelWatch,omitempty"`
	PromQL     *KanaryStatefulsetSpecValidationPromQL     `json:"promQL,omitempty"`
}

// KanaryStatefulsetSpecValidationManual defines the manual validation configuration
type KanaryStatefulsetSpecValidationManual struct {
	StatusAfterDealine KanaryStatefulsetSpecValidationManualDeadineStatus `json:"statusAfterDeadline,omitempty"`
	Status             KanaryStatefulsetSpecValidationManualStatus        `json:"status,omitempty"`
}

// KanaryStatefulsetSpecValidationManualDeadineStatus defines the validation manual deadine mode
type KanaryStatefulsetSpecValidationManualDeadineStatus string

const (
	NoneKanaryStatefulsetSpecValidationManualDeadineStatus KanaryStatefulsetSpecValidationManualDeadineStatus = "none"
	ValidKanaryStatefulsetSpecValidationManualDeadineStatus KanaryStatefulsetSpecValidationManualDeadineStatus = "valid"
	InvalidKanaryStatefulsetSpecValidationManualDeadineStatus KanaryStatefulsetSpecValidationManualDeadineStatus = "invalid"
)

// KanaryStatefulsetSpecValidationManualStatus defines the KanaryStatefulset validation status in case of manual validation.
type KanaryStatefulsetSpecValidationManualStatus string

const (
	ValidKanaryStatefulsetSpecValidationManualStatus KanaryStatefulsetSpecValidationManualStatus = "valid"
	InvalidKanaryStatefulsetSpecValidationManualStatus KanaryStatefulsetSpecValidationManualStatus = "invalid"
)

// KanaryStatefulsetSpecValidationLabelWatch defines the labelWatch validation configuration
type KanaryStatefulsetSpecValidationLabelWatch struct {
	PodInvalidationLabels *metav1.LabelSelector `json:"podInvalidationLabels,omitempty"`
	DeploymentInvalidationLabels *metav1.LabelSelector `json:"deploymentInvalidationLabels,omitempty"`
}

// KanaryStatefulsetSpecValidationPromQL defines the promQL validation configuration
type KanaryStatefulsetSpecValidationPromQL struct {
	PrometheusService string `json:"prometheusService"`
	Query             string `json:"query"` //The promQL query
	// note the AND close that prevent to return record when there is less that 70 records over the floating time window of 1m
	PodNameKey               string                    `json:"podNamekey"`   // Key to access the podName
	AllPodsQuery             bool                      `json:"allPodsQuery"` // This indicate that the query will return a result that is applicable to all pods. The pod dimension and so the PodNameKey is not taken into account. Default value is false.
	ValueInRange             *ValueInRange             `json:"valueInRange,omitempty"`
	DiscreteValueOutOfList   *DiscreteValueOutOfList   `json:"discreteValueOutOfList,omitempty"`
	ContinuousValueDeviation *ContinuousValueDeviation `json:"continuousValueDeviation,omitempty"`
}

// ValueInRange detect anomaly when the value returned is not inside the defined range
type ValueInRange struct {
	Min *float64 `json:"min"` // Min , the lower bound of the range. Default value is 0.0
	Max *float64 `json:"max"` // Max , the upper bound of the range. Default value is 1.0
}

// ContinuousValueDeviation detect anomaly when the average value for a pod is deviating from the average for the fleet of pods. If a pods does not register enough event it should not be returned by the PromQL
// The promQL should return value that are grouped by:
// 1- the podname
type ContinuousValueDeviation struct {
	//PromQL example, deviation compare to global average: (rate(solution_price_sum[1m])/rate(solution_price_count[1m]) and delta(solution_price_count[1m])>70) / scalar(sum(rate(solution_price_sum[1m]))/sum(rate(solution_price_count[1m])))
	MaxDeviationPercent *float64 `json:"maxDeviationPercent"` // MaxDeviationPercent maxDeviation computation based on % of the mean
}

// DiscreteValueOutOfList detect anomaly when the a value is not in the list with a ratio that exceed the tolerance
// The promQL should return counter that are grouped by:
// 1-the key of the value to monitor
// 2-the podname
type DiscreteValueOutOfList struct {
	//PromQL example: sum(delta(ms_rpc_count{job=\"kubernetes-pods\",run=\"foo\"}[10s])) by (code,kubernetes_pod_name)
	Key              string   `json:"key"`                  // Key for the metrics. For the previous example it will be "code"
	GoodValues       []string `json:"goodValues,omitempty"` // Good Values ["200","201"]. If empty means that BadValues should be used to do exclusion instead of inclusion.
	BadValues        []string `json:"badValues,omitempty"`  // Bad Values ["500","404"].
	TolerancePercent *uint    `json:"tolerance"`            // % of Bad values tolerated until the pod is considered out of SLA
}

// KanaryStatefulsetStatus defines the observed state of KanaryStatefulset
type KanaryStatefulsetStatus struct {
	CurrentHash string `json:"currentHash,omitempty"`
	Conditions []KanaryStatefulsetCondition `json:"conditions,omitempty"`
	Report KanaryStatefulsetStatusReport `json:"report,omitempty"`
	// todo 记录当前的进程
	KanaryTested bool `json:"kanaryTested,omitempty"`// kanary 测试批次是否已经完成
	Batches int64 `json:"batches,omitempty"` // deploy的后面的更新的批次
}

type KanaryStatefulsetStatusReport struct {
	Status     string `json:"status,omitempty"`
	Validation string `json:"validation,omitempty"`
	Scale      string `json:"scale,omitempty"`
	Traffic    string `json:"traffic,omitempty"`
}

// DeploymentTemplate is the object that describes the deployment that will be created.
type StatefulsetTemplate struct {
	metav1.TypeMeta `json:",inline"`
	
	metav1.ObjectMeta `json:"metadata,omitempty"`
	
	Spec v1beta1.StatefulSetSpec `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`
}

// KanaryStatefulsetCondition describes the state of a deployment at a certain point.
type KanaryStatefulsetCondition struct {
	
	Type KanaryStatefulsetConditionType `json:"type"`
	
	Status v1.ConditionStatus `json:"status"`
	
	LastUpdateTime metav1.Time `json:"lastUpdateTime,omitempty"`
	
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
	
	Reason string `json:"reason,omitempty"`
	
	Message string `json:"message,omitempty"`
}

// KanaryStatefulsetConditionType describes the state of a deployment at a certain point.
type KanaryStatefulsetConditionType string

// These are valid conditions of a kanarystatefulset.
const (
	
	ScheduledKanaryStatefulsetConditionType KanaryStatefulsetConditionType = "Scheduled"
	
	ActivatedKanaryStatefulsetConditionType KanaryStatefulsetConditionType = "Activated"
	
	SucceededKanaryStatefulsetConditionType KanaryStatefulsetConditionType = "Succeeded"
	
	FailedKanaryStatefulsetConditionType KanaryStatefulsetConditionType = "Failed"
	
	RunningKanaryStatefulsetConditionType KanaryStatefulsetConditionType = "Running"
	
	KanaryTestedStatefulsetConditionType KanaryStatefulsetConditionType = "KanaryTested"
	
	BatchesKanaryStatefulsetConditionType KanaryStatefulsetConditionType = "Batches"
	
	KanaryFinishedStatefulsetConditionType KanaryStatefulsetConditionType = "KanaryFinished"
	
	ErroredKanaryStatefulsetConditionType KanaryStatefulsetConditionType = "Errored"
)

// KanaryStatefulsetAnnotationKeyType corresponds to all possible Annotation Keys that can be added/updated by Kanary
type KanaryStatefulsetAnnotationKeyType string

const (
	// MD5KanaryStatefulsetAnnotationKey correspond to the annotation key for the deployment template md5 used to create the deployment.
	MD5KanaryStatefulsetAnnotationKey KanaryStatefulsetAnnotationKeyType = "kanary.k8s-operators.dev/md5"
)

const (
	KanaryStatefulsetIsKanaryLabelKey = "kanary.k8s-operators.dev/iskanary"
	
	KanaryStatefulsetKanaryNameLabelKey = "kanary.k8s-operators.dev/name"
	
	KanaryStatefulsetActivateLabelKey = "kanary.k8s-operators.dev/canary-pod"
	
	KanaryStatefulsetLabelValueTrue = "true"
	
	KanaryStatefulsetLabelValueFalse = "false"
)
