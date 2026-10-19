// Package apply writes objects with server-side apply. Each kind's Apply in
// core uses Typed with its typed client; Object writes anything else, typed or
// unstructured, through the dynamic client.
package apply

import (
	"context"
	"encoding/json"
	"fmt"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"

	"github.com/intentd/kubekit/core/types"
	"github.com/intentd/kubekit/internal/shared"
)

// Patcher is a typed client's Patch, such as
// clientset.AppsV1().Deployments(namespace).
type Patcher[T any] interface {
	Patch(ctx context.Context, name string, pt k8stypes.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (T, error)
}

// Typed writes obj with server-side apply through client, the typed client
// for obj's kind. obj must carry its apiVersion and kind.
func Typed[T any](ctx context.Context, client Patcher[T], obj types.Owner, opts types.Options) (types.Result, error) {
	body, err := json.Marshal(obj)
	if err != nil {
		return "", fmt.Errorf("marshal: %w", err)
	}
	if _, err := client.Patch(ctx, obj.GetName(), k8stypes.ApplyPatchType, body, opts.PatchOptions()); err != nil {
		return "", err
	}
	return types.Applied, nil
}

// Object writes obj with server-side apply through the dynamic client,
// finding its resource with mapper. obj may be typed or unstructured, and
// must carry its apiVersion and kind.
func Object(ctx context.Context, dyn dynamic.Interface, mapper meta.RESTMapper, obj runtime.Object, opts types.Options) (types.Result, error) {
	u, ok := obj.(*unstructured.Unstructured)
	if !ok {
		m, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
		if err != nil {
			return "", fmt.Errorf("convert to unstructured: %w", err)
		}
		u = &unstructured.Unstructured{Object: m}
	}
	res, err := shared.ResourceFor(dyn, mapper, u.GroupVersionKind(), u.GetNamespace())
	if err != nil {
		return "", err
	}
	po := opts.PatchOptions()
	if _, err := res.Apply(ctx, u.GetName(), u, metav1.ApplyOptions{FieldManager: po.FieldManager, Force: *po.Force}); err != nil {
		return "", err
	}
	return types.Applied, nil
}
