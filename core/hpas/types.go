package hpas

import "github.com/Intentd/kuberesources/core/types"

// Spec is the fully resolved HorizontalPodAutoscaler specification.
// All template expressions have been evaluated before this struct is populated.
// Passed directly to Create, Apply and Delete.
type Spec struct {
	// Name — HPA resource name. Required.
	Name string

	// Namespace — target namespace.
	Namespace string

	// ScaleTargetRef — the target workload this HPA scales.
	// Supports Deployment, ReplicaSet, StatefulSet, or any scalable resource.
	ScaleTargetRef types.ScaleTargetRef

	// MinReplicas — minimum pod replica count. Default: 1.
	MinReplicas int32

	// MaxReplicas — maximum pod replica count. Required.
	MaxReplicas int32

	// TargetCPUUtilizationPercentage — CPU utilization target (0-100).
	// When 0, no CPU metric is configured.
	TargetCPUUtilizationPercentage int32

	// Labels applied to HPA metadata.
	Labels map[string]string

	// Behavior — fully resolved scaling behavior. nil means use Kubernetes defaults.
	// When behavior.preset was declared, it is expanded here at resolve time.
	Behavior *types.HPABehavior
}
