package namespaces

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/intentd/kubewright/core/apply"
	"github.com/intentd/kubewright/core/types"
)

// Spec is the fully resolved Namespace specification.
type Spec struct {
	// Name — name. Required.
	Name string

	// Labels — applied to Namespace metadata.
	Labels map[string]string

	// Annotations — applied to Namespace metadata.
	Annotations map[string]string

	// Finalizers is an opaque list of values that must be empty to permanently remove object from storage.
	// optional
	Finalizers []string
}

// Create creates a Namespace if it does not already exist.
// Idempotent — skips if the Namespace exists.
// Owner reference set so Namespace is garbage collected when the owner is deleted.
//
// Namespaces have no meaningful spec fields that can drift after creation.
// Call Create again to bring it back if it was deleted; an existing one is
// left as it is.
func Create(ctx context.Context, kube kubernetes.Interface, owner types.Owner, spec Spec) (types.Result, error) {
	if err := validateSpec(spec); err != nil {
		return "", fmt.Errorf("namespace.Create: invalid spec: %w", err)
	}

	_, err := kube.CoreV1().Namespaces().Get(ctx, spec.Name, metav1.GetOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return "", fmt.Errorf("namespace.Create: checking existence of %q: %w", spec.Name, err)
	}
	if err == nil {
		return types.Unchanged, nil
	}

	sa := Build(owner, spec)

	_, err = kube.CoreV1().Namespaces().Create(ctx, sa, metav1.CreateOptions{})
	if err != nil {
		return "", fmt.Errorf("namespace.Create: creating namespace %q: %w", spec.Name, err)
	}

	return types.Created, nil
}

// Apply creates or updates a Namespace using Server-Side Apply.
// Sends only the fields spec sets; defaults the API server fills in are left alone.
func Apply(ctx context.Context, kube kubernetes.Interface, owner types.Owner, spec Spec, opts types.Options) (types.Result, error) {
	if err := validateSpec(spec); err != nil {
		return "", fmt.Errorf("namespace.Apply: invalid spec: %w", err)
	}

	ns := Build(owner, spec)
	ns.TypeMeta = metav1.TypeMeta{APIVersion: "v1", Kind: "Namespace"}

	if _, err := apply.Typed(ctx, kube.CoreV1().Namespaces(), ns, opts); err != nil {
		return "", fmt.Errorf("namespace.Apply: %w", err)
	}

	return types.Applied, nil
}

// Delete deletes the Namespace if it exists.
// For most cases owner references handle cleanup automatically —
// only use this when explicit cleanup control is needed.
func Delete(ctx context.Context, kube kubernetes.Interface, owner types.Owner, spec Spec) (types.Result, error) {

	err := kube.CoreV1().Namespaces().Delete(ctx, spec.Name, metav1.DeleteOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return types.NotFound, nil
		}
		return "", fmt.Errorf("namespace.Delete: deleting namespace %q: %w", spec.Name, err)
	}

	return types.Deleted, nil
}

// DeleteIfOwned deletes the Namespace if it exists and owned reports true for it.
func DeleteIfOwned(ctx context.Context, kube kubernetes.Interface, name string, owned func(metav1.Object) bool) (types.Result, error) {
	existing, err := kube.CoreV1().Namespaces().
		Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return types.NotFound, nil
		}
		return "", err
	}
	// Only delete if we own it
	if !owned(existing) {
		return types.NotOwned, nil
	}
	if err := kube.CoreV1().Namespaces().Delete(ctx, name, metav1.DeleteOptions{}); err != nil {
		return "", err
	}
	return types.Deleted, nil
}

// ── Build and its helpers ─────────────────────────────────────────────────

// Build returns the Namespace spec describes, owned by owner.
// It reads nothing from the cluster.
func Build(owner types.Owner, spec Spec) *corev1.Namespace {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:        spec.Name,
			Labels:      spec.Labels,
			Annotations: spec.Annotations,
		},
	}
	// Cluster-scoped owners can hold owner references on cluster-scoped resources.
	// Namespace-scoped owners cannot — GC would treat the namespace as orphaned and
	// delete it immediately. Fall back to label-based tracking in that case.
	if owner.GetNamespace() == "" {
		ns.OwnerReferences = types.OwnerReferences(owner)
	}
	return ns
}

func validateSpec(spec Spec) error {
	if spec.Name == "" {
		return fmt.Errorf("name is required")
	}
	return nil
}
