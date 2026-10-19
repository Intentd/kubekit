package types

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
)

// Owner is the object a resource belongs to: it gets the owner reference and
// lends its namespace when the spec names none. Any Kubernetes object
// satisfies it, typed or unstructured.
type Owner interface {
	metav1.Object
	runtime.Object
}

// OwnerReferences returns the controller owner reference to owner, which
// makes the object owner's: Kubernetes deletes it with owner.
func OwnerReferences(owner Owner) []metav1.OwnerReference {
	apiVersion := ""
	kind := ""
	if u, ok := owner.(*unstructured.Unstructured); ok {
		apiVersion = u.GetAPIVersion()
		kind = u.GetKind()
	} else {
		gvk := owner.GetObjectKind().GroupVersionKind()
		apiVersion = gvk.GroupVersion().String()
		kind = gvk.Kind
	}

	return []metav1.OwnerReference{
		{
			APIVersion:         apiVersion,
			Kind:               kind,
			Name:               owner.GetName(),
			UID:                owner.GetUID(),
			Controller:         ptr.To(true),
			BlockOwnerDeletion: ptr.To(true),
		},
	}
}

// OwnedByReference reports whether obj has an owner reference to owner, by
// UID. Pass it to DeleteIfOwned to delete only what owner owns.
func OwnedByReference(owner Owner) func(metav1.Object) bool {
	return func(obj metav1.Object) bool {
		for _, ref := range obj.GetOwnerReferences() {
			if ref.UID == owner.GetUID() {
				return true
			}
		}
		return false
	}
}

// OwnedByLabels reports whether obj carries every label in want, with the
// same values. Pass it to DeleteIfOwned for objects that cannot hold an
// owner reference, such as a cluster-scoped object with a namespaced owner:
// set the labels in the spec when you create it, and delete with the same
// labels. An empty want matches nothing, so it never deletes by accident.
func OwnedByLabels(want map[string]string) func(metav1.Object) bool {
	return func(obj metav1.Object) bool { return hasAll(obj.GetLabels(), want) }
}

// OwnedByAnnotations is OwnedByLabels for annotations: it reports whether
// obj carries every annotation in want, with the same values. Use it when
// the ownership marker should not be selectable, or is too long for a label
// value. An empty want matches nothing.
func OwnedByAnnotations(want map[string]string) func(metav1.Object) bool {
	return func(obj metav1.Object) bool { return hasAll(obj.GetAnnotations(), want) }
}

// hasAll reports whether have contains every key in want with the same
// value. An empty want is false.
func hasAll(have, want map[string]string) bool {
	if len(want) == 0 {
		return false
	}
	for k, v := range want {
		if got, ok := have[k]; !ok || got != v {
			return false
		}
	}
	return true
}
