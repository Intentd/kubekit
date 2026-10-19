// Package types holds the field types a resource spec is made of: env and
// envFrom, resource requirements, probes, pod and container security,
// volumes, rolling update, HPA behavior, network policy rules, limit range
// items, RBAC rules, labels and custom resource metadata. Each kind's Spec
// uses them, and a program that declares resources in YAML or JSON can use
// them too: they carry yaml and json tags.
//
// It also holds what every kind shares: Owner, the object resources are built
// for; Options, how Apply writes; and Result, what a verb did.
package types
