package v1alpha1

import (
	"k8s.io/api/apps/v1beta1"
	"k8s.io/api/autoscaling/v2beta1"
	v1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func init() {
	SchemeBuilder.Register(&KanaryStatefulset{}, &KanaryStatefulsetList{})
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KanaryStatefulset is the Schema for the kanarystatefulsets API
// +k8s:openapi-gen=true
type KanaryStatefulset struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KanaryStatefulsetSpec   `json:"spec,omitempty"`
	Status KanaryStatefulsetStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KanaryStatefulsetList contains a list of KanaryStatefulset
type KanaryStatefulsetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KanaryStatefulset `json:"items"`
}

// KanaryStatefulsetSpec defines the desired state of KanaryStatefulset
type KanaryStatefulsetSpec struct {
	// DeploymentName is the name of the deployment that will be updated in case of a success
	// canary deployment testing
	// if DeploymentName is empty or not define. The KanaryStatefulset will search for a Deployment with
	// same name than the KanaryStatefulset. If the deployment not exist, the deployment will be created
	// with the deployment template present in the KanaryStatefulset.
	DeploymentName string `json:"deploymentName,omitempty"`
	StatefulSetName string `json:"statefulSetName,omitempty"`
	// serviceName is the name of the service that governs the associated Deployment.
	// This service can be empty of not defined, which means that some Kanary feature will not be
	// applied on the KanaryStatefulset.
	ServiceName string `json:"serviceName,omitempty"`
	// Template  is the object that describes the deployment that will be created.
	Template DeploymentTemplate `json:"template,omitempty"`
	// Scale is the scaling configuration for the canary deployment
	Scale KanaryStatefulsetSpecScale `json:"scale,omitempty"`
	// Traffic is the scaling configuration for the canary deployment
	Traffic KanaryStatefulsetSpecTraffic `json:"traffic,omitempty"`
	// Validations is the scaling configuration for the canary deployment
	Validations KanaryStatefulsetSpecValidationList `json:"validations,omitempty"`
	// Schedule helps you to define when that canary deployment should start. RFC3339 = "2006-01-02T15:04:05Z07:00" "2006-01-02T15:04:05Z"
	Schedule string `json:"schedule,omiempty"`
}

// KanaryStatefulsetSpecScale defines the scale configuration for the canary deployment
type KanaryStatefulsetSpecScale struct {
	Static *KanaryStatefulsetSpecScaleStatic `json:"static,omitempty"`
	HPA    *HorizontalPodAutoscalerSpec     `json:"hpa,omitempty"`
}

// KanaryStatefulsetSpecScaleStatic defines the static scale configuration for the canary deployment
type KanaryStatefulsetSpecScaleStatic struct {
	// Number of desired pods. This is a pointer to distinguish between explicit
	// zero and not specified. Defaults to 1.
	// +optional
	Replicas *int32 `json:"replicas,omitempty"`
}

// HorizontalPodAutoscalerSpec describes the desired functionality of the HorizontalPodAutoscaler.
type HorizontalPodAutoscalerSpec struct {
	// minReplicas is the lower limit for the number of replicas to which the autoscaler can scale down.
	// It defaults to 1 pod.
	// +optional
	MinReplicas *int32 `json:"minReplicas,omitempty" protobuf:"varint,2,opt,name=minReplicas"`
	// maxReplicas is the upper limit for the number of replicas to which the autoscaler can scale up.
	// It cannot be less that minReplicas.
	MaxReplicas int32 `json:"maxReplicas" protobuf:"varint,3,opt,name=maxReplicas"`
	// metrics contains the specifications for which to use to calculate the
	// desired replica count (the maximum replica count across all metrics will
	// be used).  The desired replica count is calculated multiplying the
	// ratio between the target value and the current value by the current
	// number of pods.  Ergo, metrics used must decrease as the pod count is
	// increased, and vice-versa.  See the individual metric source types for
	// more information about how each type of metric must respond.
	// +optional
	Metrics []v2beta1.MetricSpec `json:"metrics,omitempty" protobuf:"bytes,4,rep,name=metrics"`
}

// KanaryStatefulsetSpecTraffic defines the traffic configuration for the canary deployment
type KanaryStatefulsetSpecTraffic struct {
	// Source defines the traffic source that targets the canary deployment pods
	Source KanaryStatefulsetSpecTrafficSource `json:"source,omitempty"`
	// KanaryService is the name of the service that will be created to target specifically
	// pods that serve the canary service version.
	// if kanaryService is empty or not define, a service name will be generated from the
	// serviceName provided in the KanaryStatefulsetSpec.
	KanaryService string `json:"kanaryService,omitempty"`
	// Mirror
	Mirror *KanaryStatefulsetSpecTrafficMirror `json:"mirror,omitempty"`
}

// KanaryStatefulsetSpecTrafficSource defines the traffic source that targets the canary deployment pods
type KanaryStatefulsetSpecTrafficSource string

const (
	// ServiceKanaryStatefulsetSpecTrafficSource means that deployment service also target the canary deployment. Normal service discovery and loadbalacing done by kubernetes will be applied.
	ServiceKanaryStatefulsetSpecTrafficSource KanaryStatefulsetSpecTrafficSource = "service"
	// KanaryServiceKanaryStatefulsetSpecTrafficSource means that a dedicated service is created to target the canary deployment pods. The canary pods do not receive traffic from the classic service.
	KanaryServiceKanaryStatefulsetSpecTrafficSource KanaryStatefulsetSpecTrafficSource = "kanary-service"
	// BothKanaryStatefulsetSpecTrafficSource means canary deployment pods are targetable thank the deployment service but also
	// with a the create kanary service.
	BothKanaryStatefulsetSpecTrafficSource KanaryStatefulsetSpecTrafficSource = "both"
	// NoneKanaryStatefulsetSpecTrafficSource means the canary deployment pods are not accessible. it can be use when the application
	// don't define any service.
	NoneKanaryStatefulsetSpecTrafficSource KanaryStatefulsetSpecTrafficSource = "none"
	// MirrorKanaryStatefulsetSpecTrafficSource means that the canary deployment pods are target by a mirror traffic. This can be done only if istio is installed.
	MirrorKanaryStatefulsetSpecTrafficSource KanaryStatefulsetSpecTrafficSource = "mirror"
)

// KanaryStatefulsetSpecTrafficMirror define the activation of mirror traffic on canary pods
type KanaryStatefulsetSpecTrafficMirror struct {
	Activate bool `json:"activate"`
}

// KanaryStatefulsetSpecValidationList define list of KanaryStatefulsetSpecValidation
type KanaryStatefulsetSpecValidationList struct {
	// InitialDelay duration after the KanaryStatefulset has started before validation checks is started.
	InitialDelay *metav1.Duration `json:"initialDelay,omitempty"`
	// ValidationPeriod validation checks duration.
	ValidationPeriod *metav1.Duration `json:"validationPeriod,omitempty"`
	// MaxIntervalPeriod max interval duration between two validation tentative
	MaxIntervalPeriod *metav1.Duration `json:"maxIntervalPeriod,omitempty"`
	// NoUpdate if set to true, the Deployment will no be updated after a succeed validation period.
	NoUpdate bool `json:"noUpdate,omitempty"`
	// Items list of KanaryStatefulsetSpecValidation
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
	// NoneKanaryStatefulsetSpecValidationManualDeadineStatus means deadline is not activated.
	NoneKanaryStatefulsetSpecValidationManualDeadineStatus KanaryStatefulsetSpecValidationManualDeadineStatus = "none"
	// ValidKanaryStatefulsetSpecValidationManualDeadineStatus means that after the validation.ValidationPeriod
	// if the validation.manual.status is not set properly the KanaryStatefulset will be considered as "valid"
	ValidKanaryStatefulsetSpecValidationManualDeadineStatus KanaryStatefulsetSpecValidationManualDeadineStatus = "valid"
	// InvalidKanaryStatefulsetSpecValidationManualDeadineStatus means that after the validation.ValidationPeriod
	// if the validation.manual.status is not set properly the KanaryStatefulset will be considered as "invalid"
	InvalidKanaryStatefulsetSpecValidationManualDeadineStatus KanaryStatefulsetSpecValidationManualDeadineStatus = "invalid"
)

// KanaryStatefulsetSpecValidationManualStatus defines the KanaryStatefulset validation status in case of manual validation.
type KanaryStatefulsetSpecValidationManualStatus string

const (
	// ValidKanaryStatefulsetSpecValidationManualStatus means that the KanaryStatefulset have been validated successfully.
	ValidKanaryStatefulsetSpecValidationManualStatus KanaryStatefulsetSpecValidationManualStatus = "valid"
	// InvalidKanaryStatefulsetSpecValidationManualStatus means that the KanaryStatefulset have been invalidated.
	InvalidKanaryStatefulsetSpecValidationManualStatus KanaryStatefulsetSpecValidationManualStatus = "invalid"
)

// KanaryStatefulsetSpecValidationLabelWatch defines the labelWatch validation configuration
type KanaryStatefulsetSpecValidationLabelWatch struct {
	// PodInvalidationLabels defines labels that should be present on the canary pods in order to invalidate
	// the canary deployment
	PodInvalidationLabels *metav1.LabelSelector `json:"podInvalidationLabels,omitempty"`
	// DeploymentInvalidationLabels defines labels that should be present on the canary deployment in order to invalidate
	// the canary deployment
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
	// CurrentHash represents the current MD5 spec deployment template hash
	CurrentHash string `json:"currentHash,omitempty"`
	// Represents the latest available observations of a kanarystatefulset's current state.
	Conditions []KanaryStatefulsetCondition `json:"conditions,omitempty"`
	// Report
	Report KanaryStatefulsetStatusReport `json:"report,omitempty"`
}

type KanaryStatefulsetStatusReport struct {
	Status     string `json:"status,omitempty"`
	Validation string `json:"validation,omitempty"`
	Scale      string `json:"scale,omitempty"`
	Traffic    string `json:"traffic,omitempty"`
}

// DeploymentTemplate is the object that describes the deployment that will be created.
type DeploymentTemplate struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Specification of the desired behavior of the Deployment.
	// +optional
	Spec v1beta1.DeploymentSpec `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`
}

// KanaryStatefulsetCondition describes the state of a deployment at a certain point.
type KanaryStatefulsetCondition struct {
	// Type of deployment condition.
	Type KanaryStatefulsetConditionType `json:"type"`
	// Status of the condition, one of True, False, Unknown.
	Status v1.ConditionStatus `json:"status"`
	// The last time this condition was updated.
	LastUpdateTime metav1.Time `json:"lastUpdateTime,omitempty"`
	// Last time the condition transitioned from one status to another.
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
	// The reason for the condition's last transition.
	Reason string `json:"reason,omitempty"`
	// A human readable message indicating details about the transition.
	Message string `json:"message,omitempty"`
}

// KanaryStatefulsetConditionType describes the state of a deployment at a certain point.
type KanaryStatefulsetConditionType string

// These are valid conditions of a kanarystatefulset.
const (
	// Activated means the KanaryStatefulset strategy is activated
	ScheduledKanaryStatefulsetConditionType KanaryStatefulsetConditionType = "Scheduled"
	// Activated means the KanaryStatefulset strategy is activated
	ActivatedKanaryStatefulsetConditionType KanaryStatefulsetConditionType = "Activated"
	// Succeeded means the KanaryStatefulset strategy succeed,
	// the deployment rolling-update is in progress or already done.
	// it means also the deployment and the canary deployment have the same version.
	SucceededKanaryStatefulsetConditionType KanaryStatefulsetConditionType = "Succeeded"
	// FailedKanaryStatefulsetConditionType is added in a kanarystatefulset when the canary deployment
	// process failed.
	FailedKanaryStatefulsetConditionType KanaryStatefulsetConditionType = "Failed"
	// RunningKanaryStatefulsetConditionType is added in a kanarystatefulset when the canary is still under validation.
	RunningKanaryStatefulsetConditionType KanaryStatefulsetConditionType = "Running"
	// DeploymentUpdated is added in a kanarystatefulset when the canary succeded and that the deployment was updated
	DeploymentUpdatedKanaryStatefulsetConditionType KanaryStatefulsetConditionType = "DeploymentUpdated"

	// ErroredKanaryStatefulsetConditionType is added in a kanarystatefulset when the canary deployment
	// process errored.
	ErroredKanaryStatefulsetConditionType KanaryStatefulsetConditionType = "Errored"
	// TrafficServiceKanaryStatefulsetConditionType means the KanaryStatefulset Traffic strategy is activated
	TrafficKanaryStatefulsetConditionType KanaryStatefulsetConditionType = "Traffic"
)

// KanaryStatefulsetAnnotationKeyType corresponds to all possible Annotation Keys that can be added/updated by Kanary
type KanaryStatefulsetAnnotationKeyType string

const (
	// MD5KanaryStatefulsetAnnotationKey correspond to the annotation key for the deployment template md5 used to create the deployment.
	MD5KanaryStatefulsetAnnotationKey KanaryStatefulsetAnnotationKeyType = "kanary.k8s-operators.dev/md5"
)

const (
	// KanaryStatefulsetIsKanaryLabelKey correspond to the label key used on a deployment to inform
	// that this instance is used in a canary deployment.
	KanaryStatefulsetIsKanaryLabelKey = "kanary.k8s-operators.dev/iskanary"
	// KanaryStatefulsetKanaryNameLabelKey correspond to the label key used on a deployment and pod to provide the KanaryStatefulset name.
	KanaryStatefulsetKanaryNameLabelKey = "kanary.k8s-operators.dev/name"
	// KanaryStatefulsetActivateLabelKey correspond to the label key used on a pod to inform that this
	// Pod instance in a canary version of the application.
	KanaryStatefulsetActivateLabelKey = "kanary.k8s-operators.dev/canary-pod"
	// KanaryStatefulsetLabelValueTrue correspond to the label value True used with several Kanary label keys.
	KanaryStatefulsetLabelValueTrue = "true"
	// KanaryStatefulsetLabelValueFalse correspond to the label value False used with several Kanary label keys.
	KanaryStatefulsetLabelValueFalse = "false"
)
