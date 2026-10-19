package shared

import (
	"fmt"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

// GVKFromFields parses apiVersion and kind into a GroupVersionKind.
func GVKFromFields(apiVersion, kind string) (schema.GroupVersionKind, error) {
	if apiVersion == "" || kind == "" {
		return schema.GroupVersionKind{}, fmt.Errorf("apiVersion and kind are required")
	}
	gv, err := schema.ParseGroupVersion(apiVersion)
	if err != nil {
		return schema.GroupVersionKind{}, err
	}
	return gv.WithKind(kind), nil
}

// ToJSONSafe converts integer types to float64, recursively, so a value from
// YAML can be set on an unstructured object: unstructured.DeepCopyJSONValue
// accepts float64 only.
func ToJSONSafe(v any) any {
	switch vv := v.(type) {
	case int:
		return float64(vv)
	case int32:
		return float64(vv)
	case int64:
		return float64(vv)
	case uint:
		return float64(vv)
	case uint32:
		return float64(vv)
	case uint64:
		return float64(vv)
	case map[string]any:
		out := make(map[string]any, len(vv))
		for k, val := range vv {
			out[k] = ToJSONSafe(val)
		}
		return out
	case []any:
		out := make([]any, len(vv))
		for i, e := range vv {
			out[i] = ToJSONSafe(e)
		}
		return out
	default:
		return vv
	}
}

// ResourceFor returns the dynamic client for gvk, scoped to namespace when the
// resource is namespaced.
func ResourceFor(dyn dynamic.Interface, mapper meta.RESTMapper, gvk schema.GroupVersionKind, namespace string) (dynamic.ResourceInterface, error) {
	mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return nil, fmt.Errorf("no REST mapping for %s: %w. Check that the CRD is installed", gvk, err)
	}
	resource := dyn.Resource(mapping.Resource)
	if mapping.Scope.Name() == meta.RESTScopeNameRoot {
		return resource, nil
	}
	return resource.Namespace(namespace), nil
}
