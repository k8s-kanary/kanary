package v1alpha1

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/api/autoscaling/v2beta1"
	corev1 "k8s.io/api/core/v1"
)

// DefaultCPUUtilization is the default value for CPU utilization, provided no other
// metrics are present.  This is here because it's used by both the v2beta1 defaulting
// logic, and the pseudo-defaulting done in v1 conversion.
const DefaultCPUUtilization = 80

// IsDefaultedKanaryStatefulset used to know if a KanaryStatefulset is already defaulted
// returns true if yes, else no
func IsDefaultedKanaryStatefulset(kd *KanaryStatefulset) bool {
	if !IsDefaultedKanaryStatefulsetSpecScale(&kd.Spec.Scale) {
		return false
	}
	if !IsDefaultedKanaryStatefulsetSpecTraffic(&kd.Spec.Traffic) {
		return false
	}
	if !IsDefaultedKanaryStatefulsetSpecValidationList(&kd.Spec.Validations) {
		return false
	}

	return true
}

// IsDefaultedKanaryStatefulsetSpecScale used to know if a KanaryStatefulsetSpecScale is already defaulted
// returns true if yes, else no
func IsDefaultedKanaryStatefulsetSpecScale(scale *KanaryStatefulsetSpecScale) bool {
	if scale.Static == nil && scale.HPA == nil {
		return false
	}

	if scale.Static != nil {
		if scale.Static.Replicas == nil {
			return false
		}
	}

	if scale.HPA != nil {
		if scale.HPA.MinReplicas == nil {
			return false
		}

		if scale.HPA.MaxReplicas == 0 {
			return false
		}

		if scale.HPA.Metrics == nil {
			return false
		}

		if len(scale.HPA.Metrics) == 0 {
			return false
		}
	}

	return true
}

// IsDefaultedKanaryStatefulsetSpecTraffic used to know if a KanaryStatefulsetSpecTraffic is already defaulted
// returns true if yes, else no
func IsDefaultedKanaryStatefulsetSpecTraffic(t *KanaryStatefulsetSpecTraffic) bool {
	if t.Source == NoneKanaryStatefulsetSpecTrafficSource ||
		t.Source == ServiceKanaryStatefulsetSpecTrafficSource ||
		t.Source == KanaryServiceKanaryStatefulsetSpecTrafficSource ||
		t.Source == BothKanaryStatefulsetSpecTrafficSource ||
		t.Source == MirrorKanaryStatefulsetSpecTrafficSource {
		return true
	}
	return false
}

// IsDefaultedKanaryStatefulsetSpecValidation used to know if a KanaryStatefulsetSpecValidation is already defaulted
// returns true if yes, else no
func IsDefaultedKanaryStatefulsetSpecValidationList(list *KanaryStatefulsetSpecValidationList) bool {
	if list.ValidationPeriod == nil {
		return false
	}

	if list.InitialDelay == nil {
		return false
	}

	if list.MaxIntervalPeriod == nil {
		return false
	}

	if list.Items == nil {
		return false
	}
	if len(list.Items) == 0 {
		return false
	}

	for _, v := range list.Items {
		if isInit := IsDefaultedKanaryStatefulsetSpecValidation(&v); !isInit {
			return false
		}
	}

	return true
}

// IsDefaultedKanaryStatefulsetSpecValidation used to know if a KanaryStatefulsetSpecValidation is already defaulted
// returns true if yes, else no
func IsDefaultedKanaryStatefulsetSpecValidation(v *KanaryStatefulsetSpecValidation) bool {
	if v.Manual == nil && v.LabelWatch == nil && v.PromQL == nil {
		return false
	}

	if v.Manual != nil {
		if !(v.Manual.StatusAfterDealine == NoneKanaryStatefulsetSpecValidationManualDeadineStatus ||
			v.Manual.StatusAfterDealine == ValidKanaryStatefulsetSpecValidationManualDeadineStatus ||
			v.Manual.StatusAfterDealine == InvalidKanaryStatefulsetSpecValidationManualDeadineStatus) {
			return false
		}
	}

	if v.PromQL != nil {
		if !isDefaultedKanaryStatefulsetSpecValidationPromQL(v.PromQL) {
			return false
		}
	}

	return true
}

func isDefaultedKanaryStatefulsetSpecValidationPromQL(pq *KanaryStatefulsetSpecValidationPromQL) bool {
	if pq.PrometheusService == "" {
		return false
	}
	if pq.PodNameKey == "" {
		return false
	}
	if pq.DiscreteValueOutOfList != nil && !isDefaultedKanaryStatefulsetSpecValidationPromQLDiscrete(pq.DiscreteValueOutOfList) {
		return false
	}
	if pq.ContinuousValueDeviation != nil && !isDefaultedKanaryStatefulsetSpecValidationPromQLContinuous(pq.ContinuousValueDeviation) {
		return false
	}
	if pq.ValueInRange != nil && !isDefaultedKanaryStatefulsetSpecValidationPromQLValueInRange(pq.ValueInRange) {
		return false
	}

	return true
}
func isDefaultedKanaryStatefulsetSpecValidationPromQLValueInRange(c *ValueInRange) bool {
	return c.Min != nil && c.Max != nil
}

func isDefaultedKanaryStatefulsetSpecValidationPromQLContinuous(c *ContinuousValueDeviation) bool {
	return c.MaxDeviationPercent != nil
}

func isDefaultedKanaryStatefulsetSpecValidationPromQLDiscrete(d *DiscreteValueOutOfList) bool {
	return d.TolerancePercent != nil
}

// DefaultKanaryStatefulset used to default a KanaryStatefulset
// return a list of errors in case of unvalid fields.
func DefaultKanaryStatefulset(kd *KanaryStatefulset) *KanaryStatefulset {
	defaultedKD := kd.DeepCopy()
	defaultKanaryStatefulsetSpec(&defaultedKD.Spec)
	return defaultedKD
}

// defaultKanaryStatefulsetSpec used to default a KanaryStatefulsetSpec
// return a list of errors in case of unvalid Spec.
func defaultKanaryStatefulsetSpec(spec *KanaryStatefulsetSpec) {
	defaultKanaryStatefulsetSpecScale(&spec.Scale)
	defaultKanaryStatefulsetSpecTraffic(&spec.Traffic)
	defaultKanaryStatefulsetSpecValidationList(&spec.Validations)
}

func defaultKanaryStatefulsetSpecScale(s *KanaryStatefulsetSpecScale) {
	if s.Static == nil && s.HPA == nil {
		s.Static = &KanaryStatefulsetSpecScaleStatic{}
	}
	if s.Static != nil {
		defaultKanaryStatefulsetSpecScaleStatic(s.Static)
	}
	if s.HPA != nil {
		defaultKanaryStatefulsetSpecScaleHPA(s.HPA)
	}
}

// defaultKanaryStatefulsetSpecScaleHPA used to default HorizontalPodAutoscaler spec
func defaultKanaryStatefulsetSpecScaleHPA(s *HorizontalPodAutoscalerSpec) {
	if s.MinReplicas == nil {
		s.MinReplicas = NewInt32(1)
	}
	if s.MaxReplicas == 0 {
		s.MaxReplicas = int32(10)
	}
	if s.Metrics == nil {
		s.Metrics = []v2beta1.MetricSpec{
			{
				Type: v2beta1.ResourceMetricSourceType,
				Resource: &v2beta1.ResourceMetricSource{
					Name:                     corev1.ResourceCPU,
					TargetAverageUtilization: NewInt32(DefaultCPUUtilization),
				},
			},
		}
	}
}

func defaultKanaryStatefulsetSpecScaleStatic(s *KanaryStatefulsetSpecScaleStatic) {
	if s.Replicas == nil {
		s.Replicas = NewInt32(1)
	}
}

func defaultKanaryStatefulsetSpecTraffic(t *KanaryStatefulsetSpecTraffic) {
	if !(t.Source == NoneKanaryStatefulsetSpecTrafficSource ||
		t.Source == ServiceKanaryStatefulsetSpecTrafficSource ||
		t.Source == KanaryServiceKanaryStatefulsetSpecTrafficSource ||
		t.Source == BothKanaryStatefulsetSpecTrafficSource ||
		t.Source == MirrorKanaryStatefulsetSpecTrafficSource) {
		t.Source = NoneKanaryStatefulsetSpecTrafficSource
	}

	if t.Mirror != nil {
		defaultKanaryStatefulsetSpecScaleTrafficMirror(t.Mirror)
	}
}

func defaultKanaryStatefulsetSpecScaleTrafficMirror(t *KanaryStatefulsetSpecTrafficMirror) {
	// TODO nothing todo for the moment
}

func defaultKanaryStatefulsetSpecValidationList(list *KanaryStatefulsetSpecValidationList) {
	if list == nil {
		return
	}
	if list.ValidationPeriod == nil {
		list.ValidationPeriod = &metav1.Duration{
			Duration: 15 * time.Minute,
		}
	}
	if list.InitialDelay == nil {
		list.InitialDelay = &metav1.Duration{
			Duration: 0 * time.Minute,
		}
	}
	if list.MaxIntervalPeriod == nil {
		list.MaxIntervalPeriod = &metav1.Duration{
			Duration: 20 * time.Second,
		}
	}

	if list.Items == nil || len(list.Items) == 0 {
		list.Items = []KanaryStatefulsetSpecValidation{
			{},
		}
	}
	for id, value := range list.Items {
		defaultKanaryStatefulsetSpecValidation(&value)
		list.Items[id] = value
	}
}

func defaultKanaryStatefulsetSpecValidation(v *KanaryStatefulsetSpecValidation) {
	if v.Manual == nil && v.LabelWatch == nil && v.PromQL == nil {
		defaultKanaryStatefulsetSpecScaleValidationManual(v)
	}
	if v.Manual != nil {
		if v.Manual.StatusAfterDealine == "" {
			v.Manual.StatusAfterDealine = NoneKanaryStatefulsetSpecValidationManualDeadineStatus
		}
	}
	if v.PromQL != nil {
		defaultKanaryStatefulsetSpecValidationPromQL(v.PromQL)

	}
}
func defaultKanaryStatefulsetSpecValidationPromQL(pq *KanaryStatefulsetSpecValidationPromQL) {
	if pq.PrometheusService == "" {
		pq.PrometheusService = "prometheus:9090"
	}
	if pq.PodNameKey == "" {
		pq.PodNameKey = "pod"
	}
	if pq.ContinuousValueDeviation != nil {
		defaultKanaryStatefulsetSpecValidationPromQLContinuous(pq.ContinuousValueDeviation)
	}
	if pq.DiscreteValueOutOfList != nil {
		defaultKanaryStatefulsetSpecValidationPromQLDiscreteValueOutOfList(pq.DiscreteValueOutOfList)
	}
	if pq.ValueInRange != nil {
		defaultKanaryStatefulsetSpecValidationPromQLValueInRange(pq.ValueInRange)
	}
}
func defaultKanaryStatefulsetSpecValidationPromQLValueInRange(c *ValueInRange) {
	if c.Min == nil {
		c.Min = NewFloat64(0)
	}
	if c.Max == nil {
		c.Max = NewFloat64(1)
	}
}
func defaultKanaryStatefulsetSpecValidationPromQLContinuous(c *ContinuousValueDeviation) {
	if c.MaxDeviationPercent == nil {
		c.MaxDeviationPercent = NewFloat64(10)
	}
}
func defaultKanaryStatefulsetSpecValidationPromQLDiscreteValueOutOfList(d *DiscreteValueOutOfList) {
	if d.TolerancePercent == nil {
		d.TolerancePercent = NewUInt(0)
	}
}
func defaultKanaryStatefulsetSpecScaleValidationManual(v *KanaryStatefulsetSpecValidation) {
	v.Manual = &KanaryStatefulsetSpecValidationManual{
		StatusAfterDealine: NoneKanaryStatefulsetSpecValidationManualDeadineStatus,
	}
}

// NewInt32 returns new int32 pointer instance
func NewInt32(i int32) *int32 {
	return &i
}

// NewUInt returns new uint pointer instance
func NewUInt(i uint) *uint {
	return &i
}

// NewFloat64 return a pointer to a float64
func NewFloat64(val float64) *float64 {
	return &val
}
