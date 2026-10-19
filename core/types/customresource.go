package types

// CustomResourceMetadata is the subset of metav1.ObjectMeta a custom
// resource spec sets.
//
//   - Name is required.
//   - Namespace is required for a namespaced kind and must be empty for a
//     cluster-scoped one.
//   - Labels and Annotations must follow Kubernetes key and value rules.
type CustomResourceMetadata struct {
	// Name is the resource name. Must be a valid Kubernetes name.
	Name string `json:"name" yaml:"name"`

	// Namespace is optional in the struct but required for namespaced CRDs.
	Namespace string `json:"namespace,omitempty" yaml:"namespace,omitempty"`

	// Namespaced indicates whether the target GVK is intended to be namespaced.
	// Three states are useful and intentionally supported:
	//   - nil: unspecified; treat as namespaced, or look the scope up with
	//          discovery.
	//   - pointer to true: explicitly namespaced; metadata.namespace is required.
	//   - pointer to false: explicitly cluster-scoped; metadata.namespace must be empty.
	//
	// Defaulting/validation contract:
	// - Validation should treat nil as "namespaced by default" for user ergonomics.
	// - At runtime the reconciler should still consult discovery/RESTMapper to
	//   confirm the actual CRD scope and register the GVK with activateMissing if
	//   the CRD is not present yet.
	// - If the user sets Namespaced=false but discovery shows the CRD is namespaced,
	//   the reconciler should surface a clear validation error.
	Namespaced *bool `json:"namespaced,omitempty" yaml:"namespaced,omitempty"`

	// Labels are used for selection, ownership, and children tracking.
	// Keys and values must conform to Kubernetes label syntax.
	Labels Labels `json:"labels,omitempty" yaml:"labels,omitempty"`

	// Annotations are free-form metadata. Keys must conform to Kubernetes
	// annotation key syntax. Values are arbitrary strings.
	Annotations Labels `json:"annotations,omitempty" yaml:"annotations,omitempty"`
}
