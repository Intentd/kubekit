package roles

import (
	"context"
	"fmt"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/intentd/kubewright/core/apply"
	"github.com/intentd/kubewright/core/types"
	"github.com/intentd/kubewright/internal/shared"
)

// Spec is the fully resolved Role specification.
type Spec struct {
	Name      string
	Namespace string
	Labels    map[string]string
	Rules     []rbacv1.PolicyRule
}

// Create creates a Role if it does not already exist.
// Idempotent — skips if the Role already exists.
// Owner reference ensures cleanup when the owner is deleted.
func Create(ctx context.Context, kube kubernetes.Interface, owner types.Owner, spec Spec) (types.Result, error) {
	if err := validateSpec(spec); err != nil {
		return "", fmt.Errorf("role.Create: invalid spec: %w", err)
	}

	namespace := shared.ResolveNamespace(owner, spec.Namespace)

	_, err := kube.RbacV1().Roles(namespace).Get(ctx, spec.Name, metav1.GetOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return "", fmt.Errorf("role.Create: checking existence of %q: %w", spec.Name, err)
	}
	if err == nil {
		return types.Unchanged, nil
	}

	role := Build(owner, spec)

	_, err = kube.RbacV1().Roles(namespace).Create(ctx, role, metav1.CreateOptions{})
	if err != nil {
		return "", fmt.Errorf("role.Create: creating role %q in %q: %w", spec.Name, namespace, err)
	}

	return types.Created, nil
}

// Apply creates or updates a Role using Server-Side Apply.
// Sends only the fields spec sets; defaults the API server fills in are left alone.
func Apply(ctx context.Context, kube kubernetes.Interface, owner types.Owner, spec Spec, opts types.Options) (types.Result, error) {
	namespace := shared.ResolveNamespace(owner, spec.Namespace)

	role := Build(owner, spec)
	role.TypeMeta = metav1.TypeMeta{APIVersion: "rbac.authorization.k8s.io/v1", Kind: "Role"}

	if _, err := apply.Typed(ctx, kube.RbacV1().Roles(namespace), role, opts); err != nil {
		return "", fmt.Errorf("role.Apply: %w", err)
	}

	return types.Applied, nil
}

// Delete deletes the Role if it exists.
func Delete(ctx context.Context, kube kubernetes.Interface, owner types.Owner, spec Spec) (types.Result, error) {
	namespace := shared.ResolveNamespace(owner, spec.Namespace)

	err := kube.RbacV1().Roles(namespace).Delete(ctx, spec.Name, metav1.DeleteOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return types.NotFound, nil
		}
		return "", fmt.Errorf("role.Delete: deleting role %q in %q: %w", spec.Name, namespace, err)
	}

	return types.Deleted, nil
}

// DeleteIfOwned deletes the Role if it exists and owned reports true for it.
func DeleteIfOwned(ctx context.Context, kube kubernetes.Interface, name, namespace string, owned func(metav1.Object) bool) (types.Result, error) {
	existing, err := kube.RbacV1().Roles(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return types.NotFound, nil
		}
		return "", err
	}
	if !owned(existing) {
		return types.NotOwned, nil
	}
	if err := kube.RbacV1().Roles(namespace).Delete(ctx, name, metav1.DeleteOptions{}); err != nil {
		return "", err
	}
	return types.Deleted, nil
}

// ── Build and its helpers ─────────────────────────────────────────────────

// Build returns the Role spec describes, owned by owner, in spec.Namespace or else the owner's namespace.
// It reads nothing from the cluster.
func Build(owner types.Owner, spec Spec) *rbacv1.Role {
	namespace := shared.ResolveNamespace(owner, spec.Namespace)
	return &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:            spec.Name,
			Namespace:       namespace,
			Labels:          spec.Labels,
			OwnerReferences: types.OwnerReferences(owner),
		},
		Rules: spec.Rules,
	}
}

func validateSpec(spec Spec) error {
	if spec.Name == "" {
		return fmt.Errorf("name is required")
	}
	return nil
}
