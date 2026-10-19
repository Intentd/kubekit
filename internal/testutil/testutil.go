// Package testutil holds what the kind packages' tests share: an owner to
// build for, and a check of the verbs' results against a fake clientset.
package testutil

import (
	"context"
	"testing"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/intentd/kubewright/core/types"
)

// OwnerUID is Owner's UID.
const OwnerUID = "11111111-2222-3333-4444-555555555555"

// Owner returns the object the tests build for: an App named "app" in
// namespace "team".
func Owner() *unstructured.Unstructured {
	u := &unstructured.Unstructured{}
	u.SetAPIVersion("demo.example.com/v1")
	u.SetKind("App")
	u.SetName("app")
	u.SetNamespace("team")
	u.SetUID(OwnerUID)
	return u
}

// Lifecycle is one kind's verbs, bound to a spec, for Check.
type Lifecycle struct {
	Create func(ctx context.Context, cs kubernetes.Interface) (types.Result, error)
	// Apply is nil for kinds without one.
	Apply func(ctx context.Context, cs kubernetes.Interface) (types.Result, error)
	// ApplyResult is what Apply reports for an object Create just made:
	// Applied unless set.
	ApplyResult types.Result
	Delete      func(ctx context.Context, cs kubernetes.Interface) (types.Result, error)
	// DeleteIfOwned is nil for kinds without one.
	DeleteIfOwned func(ctx context.Context, cs kubernetes.Interface, owned func(metav1.Object) bool) (types.Result, error)
	// Get reads the object back.
	Get func(ctx context.Context, cs kubernetes.Interface) (metav1.Object, error)
}

// Check runs the verbs against a fake clientset and checks each result:
// Create is Created then Unchanged, Delete is Deleted then
// NotFound, and DeleteIfOwned leaves what the predicate does not own.
// Apply is Applied, or ApplyResult when set.
func (l Lifecycle) Check(t *testing.T) {
	t.Helper()
	ctx := context.Background()
	cs := fake.NewClientset()
	ours := func(metav1.Object) bool { return true }
	notOurs := func(metav1.Object) bool { return false }

	res, err := l.Create(ctx, cs)
	want(t, "first Create", res, err, types.Created)
	exists(t, ctx, cs, l.Get, true)

	res, err = l.Create(ctx, cs)
	want(t, "second Create", res, err, types.Unchanged)

	if l.Apply != nil {
		applied := l.ApplyResult
		if applied == "" {
			applied = types.Applied
		}
		res, err = l.Apply(ctx, cs)
		want(t, "Apply", res, err, applied)
		exists(t, ctx, cs, l.Get, true)
	}

	if l.DeleteIfOwned != nil {
		res, err = l.DeleteIfOwned(ctx, cs, notOurs)
		want(t, "DeleteIfOwned of another owner's", res, err, types.NotOwned)
		exists(t, ctx, cs, l.Get, true)
	}

	res, err = l.Delete(ctx, cs)
	want(t, "Delete", res, err, types.Deleted)
	exists(t, ctx, cs, l.Get, false)

	res, err = l.Delete(ctx, cs)
	want(t, "second Delete", res, err, types.NotFound)

	if l.DeleteIfOwned != nil {
		res, err = l.Create(ctx, cs)
		want(t, "Create again", res, err, types.Created)
		res, err = l.DeleteIfOwned(ctx, cs, ours)
		want(t, "DeleteIfOwned of ours", res, err, types.Deleted)
		exists(t, ctx, cs, l.Get, false)
		res, err = l.DeleteIfOwned(ctx, cs, ours)
		want(t, "DeleteIfOwned of a deleted one", res, err, types.NotFound)
	}
}

// want fails the test unless a verb returned expected with no error.
func want(t *testing.T, step string, got types.Result, err error, expected types.Result) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s: %v", step, err)
	}
	if got != expected {
		t.Fatalf("%s: result %q, want %q", step, got, expected)
	}
}

func exists(t *testing.T, ctx context.Context, cs kubernetes.Interface, get func(context.Context, kubernetes.Interface) (metav1.Object, error), want bool) {
	t.Helper()
	_, err := get(ctx, cs)
	switch {
	case want && err != nil:
		t.Fatalf("object should exist: %v", err)
	case !want && err == nil:
		t.Fatal("object should be gone")
	case !want && !k8serrors.IsNotFound(err):
		t.Fatalf("reading a deleted object: %v", err)
	}
}

// Owned reports whether obj has one owner reference, to Owner.
func Owned(obj metav1.Object) bool {
	refs := obj.GetOwnerReferences()
	return len(refs) == 1 && string(refs[0].UID) == OwnerUID && refs[0].Kind == "App" && refs[0].Name == "app"
}
