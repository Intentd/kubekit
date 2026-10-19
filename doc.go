// Package kubewright builds and writes Kubernetes resources for an owner
// object: from a plain Go spec to a Deployment, Service, Secret or another
// built-in kind, with owner references set so the cluster removes them with
// their owner. It is a library, not a framework: no controller, no reconcile
// loop, no logging. You call it from your own.
//
// Each kind is its own package under core, with the same verbs:
//
//   - Build returns the object for a spec and does no I/O.
//   - Create creates it if it does not exist.
//   - Apply writes it with server-side apply.
//   - Delete deletes it; DeleteIfOwned deletes it only if a predicate says
//     the caller owns it.
//
// Every verb takes a kubernetes.Interface (customresources takes a
// dynamic.Interface and a RESTMapper) and returns a types.Result, such as
// Created, Unchanged or NotOwned, with a nil error for every expected
// outcome. Calling a verb twice is safe: an object that already exists is
// Unchanged, and one already gone is NotFound.
//
// The packages:
//
//   - core/<kind>: one package per kind, such as core/deployments
//   - core/types: the field types specs are made of, Owner, Options and Result
//   - core/apply: the one server-side apply every Apply writes through
//   - presets: named field settings ("medium", "restricted", "safe") that a
//     spec refers to instead of spelling them out
package kubewright
