package clusterroles

import (
	"context"
	"testing"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/intentd/kubewright/core/types"
	"github.com/intentd/kubewright/internal/testutil"
)

func spec() Spec {
	return Spec{Name: "web", Rules: []rbacv1.PolicyRule{{APIGroups: []string{""}, Resources: []string{"pods"}, Verbs: []string{"get"}}}}
}

func TestBuild(t *testing.T) {
	o := Build(testutil.Owner(), spec())

	if o.Name != "web" {
		t.Errorf("name = %q, want web", o.Name)
	}
	// Cluster-scoped: a namespaced owner cannot own it, so no owner reference.
	if len(o.OwnerReferences) != 0 {
		t.Errorf("owner references = %+v, want none", o.OwnerReferences)
	}
	if len(o.Rules) != 1 || o.Rules[0].Resources[0] != "pods" {
		t.Errorf("rules = %+v, want the one rule", o.Rules)
	}
}

func TestLifecycle(t *testing.T) {
	owner, s := testutil.Owner(), spec()
	testutil.Lifecycle{
		Create: func(ctx context.Context, cs kubernetes.Interface) (types.Result, error) {
			return Create(ctx, cs, owner, s)
		},
		Apply: func(ctx context.Context, cs kubernetes.Interface) (types.Result, error) {
			return Apply(ctx, cs, owner, s, types.Options{})
		},
		Delete: func(ctx context.Context, cs kubernetes.Interface) (types.Result, error) {
			return Delete(ctx, cs, owner, s)
		},
		DeleteIfOwned: func(ctx context.Context, cs kubernetes.Interface, owned func(metav1.Object) bool) (types.Result, error) {
			return DeleteIfOwned(ctx, cs, "web", owned)
		},
		Get: func(ctx context.Context, cs kubernetes.Interface) (metav1.Object, error) {
			return cs.RbacV1().ClusterRoles().Get(ctx, "web", metav1.GetOptions{})
		},
	}.Check(t)
}

// A namespaced owner can't hold an owner reference on a ClusterRole, so the
// role is tracked by a label set in the spec and deleted by the same label.
func TestDeleteIfOwned_NamespacedOwnerByLabels(t *testing.T) {
	ctx := context.Background()
	cs := fake.NewClientset()
	owner := testutil.Owner() // namespaced: "app" in "team"
	mine := map[string]string{"app.example.com/owner": "team.app"}

	s := spec()
	s.Labels = mine
	if _, err := Create(ctx, cs, owner, s); err != nil {
		t.Fatal(err)
	}

	res, err := DeleteIfOwned(ctx, cs, "web", types.OwnedByReference(owner))
	if err != nil || res != types.NotOwned {
		t.Fatalf("by reference = %q, %v; want not owned: the role has no owner reference", res, err)
	}
	res, err = DeleteIfOwned(ctx, cs, "web", types.OwnedByLabels(map[string]string{"app.example.com/owner": "team.other"}))
	if err != nil || res != types.NotOwned {
		t.Fatalf("by another owner's label = %q, %v; want not owned", res, err)
	}
	res, err = DeleteIfOwned(ctx, cs, "web", types.OwnedByLabels(mine))
	if err != nil || res != types.Deleted {
		t.Fatalf("by our label = %q, %v; want deleted", res, err)
	}
}

func TestBuild_Metadata(t *testing.T) {
	s := spec()
	s.Labels = map[string]string{"app.example.com/owner": "team.app"}
	s.Annotations = map[string]string{"app.example.com/owned-by": "team/app"}
	o := Build(testutil.Owner(), s)

	if o.Labels["app.example.com/owner"] != "team.app" {
		t.Errorf("labels = %v, want the spec's", o.Labels)
	}
	if o.Annotations["app.example.com/owned-by"] != "team/app" {
		t.Errorf("annotations = %v, want the spec's", o.Annotations)
	}
}
