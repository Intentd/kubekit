package apply

import (
	"context"
	"strings"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8stypes "k8s.io/apimachinery/pkg/types"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/fake"
	clienttesting "k8s.io/client-go/testing"

	"github.com/Intentd/kuberesources/core/types"
)

func TestTyped(t *testing.T) {
	ctx := context.Background()
	cs := fake.NewClientset()
	cm := &corev1.ConfigMap{
		TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "ConfigMap"},
		ObjectMeta: metav1.ObjectMeta{Name: "web", Namespace: "team"},
		Data:       map[string]string{"k": "v1"},
	}

	for _, v := range []string{"v1", "v2"} {
		cm.Data["k"] = v
		res, err := Typed(ctx, cs.CoreV1().ConfigMaps("team"), cm, types.Options{FieldManager: "test"})
		if err != nil || res != types.Applied {
			t.Fatalf("Typed = %q, %v; want applied", res, err)
		}
		got, err := cs.CoreV1().ConfigMaps("team").Get(ctx, "web", metav1.GetOptions{})
		if err != nil || got.Data["k"] != v {
			t.Fatalf("after applying %s: %v, %v", v, got.Data, err)
		}
	}
}

func TestObject(t *testing.T) {
	ctx := context.Background()
	gvr := schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}
	gvk := schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "Deployment"}
	mapper := meta.NewDefaultRESTMapper([]schema.GroupVersion{gvk.GroupVersion()})
	mapper.Add(gvk, meta.RESTScopeNamespace)

	// The fake dynamic client cannot do server-side apply for unstructured
	// objects, so record what Object sends instead.
	dyn := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(),
		map[schema.GroupVersionResource]string{gvr: "DeploymentList"})
	var sent []clienttesting.PatchActionImpl
	dyn.PrependReactor("patch", "deployments", func(a clienttesting.Action) (bool, runtime.Object, error) {
		sent = append(sent, a.(clienttesting.PatchActionImpl))
		return true, &unstructured.Unstructured{}, nil
	})

	typed := &appsv1.Deployment{
		TypeMeta:   metav1.TypeMeta{APIVersion: "apps/v1", Kind: "Deployment"},
		ObjectMeta: metav1.ObjectMeta{Name: "typed", Namespace: "team"},
	}
	untyped := &unstructured.Unstructured{}
	untyped.SetGroupVersionKind(gvk)
	untyped.SetName("untyped")
	untyped.SetNamespace("team")

	for _, obj := range []runtime.Object{typed, untyped} {
		res, err := Object(ctx, dyn, mapper, obj, types.Options{FieldManager: "test", Force: true})
		if err != nil || res != types.Applied {
			t.Fatalf("Object(%T) = %q, %v; want applied", obj, res, err)
		}
	}

	if len(sent) != 2 {
		t.Fatalf("%d patches sent, want 2", len(sent))
	}
	for i, name := range []string{"typed", "untyped"} {
		a := sent[i]
		if a.GetPatchType() != k8stypes.ApplyPatchType || a.GetResource() != gvr || a.GetNamespace() != "team" || a.GetName() != name {
			t.Errorf("patch %d = %s %s %s/%s, want an apply of %s team/%s", i, a.GetPatchType(), a.GetResource(), a.GetNamespace(), a.GetName(), gvr, name)
		}
		if !strings.Contains(string(a.GetPatch()), `"kind":"Deployment"`) {
			t.Errorf("patch %d body = %s, want the object with its kind", i, a.GetPatch())
		}
	}
}

func TestObject_NoMapping(t *testing.T) {
	dyn := dynamicfake.NewSimpleDynamicClient(runtime.NewScheme())
	mapper := meta.NewDefaultRESTMapper(nil)
	u := &unstructured.Unstructured{}
	u.SetAPIVersion("example.com/v1")
	u.SetKind("Unknown")
	u.SetName("x")
	if _, err := Object(context.Background(), dyn, mapper, u, types.Options{}); err == nil {
		t.Error("an object with no REST mapping should be an error")
	}
}
