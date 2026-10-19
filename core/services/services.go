package services

import (
	"context"
	"fmt"
	"maps"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"

	"github.com/intentd/kubekit/core/apply"
	"github.com/intentd/kubekit/core/types"
	"github.com/intentd/kubekit/internal/shared"
)

// Create creates a Service owned by owner if it does not already exist.
// Idempotent — if the Service exists, does nothing and returns nil.
// Owner reference is set so the Service is garbage collected when the owner is deleted.
func Create(ctx context.Context, kube kubernetes.Interface, owner types.Owner, spec Spec) (types.Result, error) {
	if err := validateSpec(spec); err != nil {
		return "", fmt.Errorf("service.Create: invalid spec: %w", err)
	}

	namespace := shared.ResolveNamespace(owner, spec.Namespace)

	_, err := kube.CoreV1().Services(namespace).Get(ctx, spec.Name, metav1.GetOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return "", fmt.Errorf("service.Create: checking existence of %q: %w", spec.Name, err)
	}
	if err == nil {
		return types.Unchanged, nil
	}

	svc := Build(owner, spec)

	_, err = kube.CoreV1().Services(namespace).Create(ctx, svc, metav1.CreateOptions{})
	if err != nil {
		return "", fmt.Errorf("service.Create: creating service %q in %q: %w", spec.Name, namespace, err)
	}

	return types.Created, nil
}

// Apply creates or updates a Service using Server-Side Apply.
// Sends only the fields spec sets; defaults the API server fills in are left alone.
func Apply(ctx context.Context, kube kubernetes.Interface, owner types.Owner, spec Spec, opts types.Options) (types.Result, error) {
	if err := validateSpec(spec); err != nil {
		return "", fmt.Errorf("service.Apply: invalid spec: %w", err)
	}

	namespace := shared.ResolveNamespace(owner, spec.Namespace)

	svc := Build(owner, spec)
	svc.TypeMeta = metav1.TypeMeta{APIVersion: "v1", Kind: "Service"}

	if _, err := apply.Typed(ctx, kube.CoreV1().Services(namespace), svc, opts); err != nil {
		return "", fmt.Errorf("service.Apply: %w", err)
	}

	return types.Applied, nil
}

// Delete deletes the Service if it exists.
func Delete(ctx context.Context, kube kubernetes.Interface, owner types.Owner, spec Spec) (types.Result, error) {
	namespace := shared.ResolveNamespace(owner, spec.Namespace)

	err := kube.CoreV1().Services(namespace).Delete(ctx, spec.Name, metav1.DeleteOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return types.NotFound, nil
		}
		return "", fmt.Errorf("service.Delete: deleting service %q in %q: %w", spec.Name, namespace, err)
	}

	return types.Deleted, nil
}

// DeleteIfOwned deletes the Service if it exists and owned reports true for it.
func DeleteIfOwned(ctx context.Context, kube kubernetes.Interface, name, namespace string, owned func(metav1.Object) bool) (types.Result, error) {
	existing, err := kube.CoreV1().Services(namespace).
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
	if err := kube.CoreV1().Services(namespace).Delete(ctx, name, metav1.DeleteOptions{}); err != nil {
		return "", err
	}
	return types.Deleted, nil
}

// ── Build and its helpers ─────────────────────────────────────────────────

// Build returns the Service spec describes, owned by owner, in spec.Namespace or else the owner's namespace.
// It reads nothing from the cluster.
func Build(owner types.Owner, spec Spec) *corev1.Service {
	namespace := shared.ResolveNamespace(owner, spec.Namespace)
	svcType := corev1.ServiceTypeClusterIP
	switch spec.Type {
	case "NodePort":
		svcType = corev1.ServiceTypeNodePort
	case "LoadBalancer":
		svcType = corev1.ServiceTypeLoadBalancer
	}

	if spec.Headless {
		svcType = corev1.ClusterIPNone
	}

	proto := corev1.ProtocolTCP
	switch strings.ToUpper(spec.Protocol) {
	case "UDP":
		proto = corev1.ProtocolUDP
	case "SCTP":
		proto = corev1.ProtocolSCTP
	case "", "TCP":
		proto = corev1.ProtocolTCP
	}

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:            spec.Name,
			Namespace:       namespace,
			Labels:          spec.Labels,
			OwnerReferences: types.OwnerReferences(owner),
		},
		Spec: corev1.ServiceSpec{
			Type:     svcType,
			Selector: maps.Clone(spec.Selector),
			Ports: []corev1.ServicePort{
				{
					Port:       spec.Port,
					TargetPort: intstr.FromInt(int(spec.TargetPort)),
					Protocol:   proto,
				},
			},
		},
	}
}

func validateSpec(spec Spec) error {
	var missing []string
	if spec.Name == "" {
		missing = append(missing, "name")
	}
	if spec.Port == 0 {
		missing = append(missing, "port")
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required fields: %v", missing)
	}
	return nil
}
