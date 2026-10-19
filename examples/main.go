// Command examples shows kubewright building the resources an App owns: presets
// expanded into fields, then each verb's result, then deletion that leaves
// another owner's objects alone. Run it from the repository root:
//
//	go run ./examples
//
// It runs against client-go's fake clientset, so it needs no cluster. In a
// real program the clientset comes from your kubeconfig or the in-cluster
// config, and the owner is your own custom resource.
package main

import (
	"context"
	"fmt"
	"log"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/intentd/kubewright/core/deployments"
	"github.com/intentd/kubewright/core/services"
	"github.com/intentd/kubewright/core/types"
	"github.com/intentd/kubewright/presets"
)

func main() {
	ctx := context.Background()
	kube := fake.NewClientset()
	app := owner("shop", "team")

	// 1. Presets: the built-ins, plus a size of our own.
	reg := presets.Builtin().With(presets.Registry{
		Resources: []presets.Resource{{
			Name:     "shop-api",
			Requests: map[string]string{"cpu": "300m", "memory": "384Mi"},
			Limits:   map[string]string{"cpu": "1", "memory": "768Mi"},
		}},
	})

	resources, err := presets.ExpandResources(&types.ResourceRequirements{Preset: "shop-api"}, reg)
	must(err)
	rollout, err := presets.ExpandRollingUpdate(&types.RollingUpdateBehavior{Preset: "safe"}, reg)
	must(err)

	fmt.Println("Presets:")
	fmt.Printf("  shop-api  requests cpu %s, memory %s\n", resources.Requests["cpu"], resources.Requests["memory"])
	fmt.Printf("  safe      maxSurge %s, maxUnavailable %s\n", rollout.MaxSurge, rollout.MaxUnavailable)
	if _, err := presets.ExpandResources(&types.ResourceRequirements{Preset: "shop-apii"}, reg); err != nil {
		fmt.Printf("  a typo    %v\n", err)
	}

	// 2. The verbs: each call says what it did.
	web := deployments.Spec{
		Name:          "web",
		Image:         "nginx:1.27",
		Replicas:      2,
		Port:          8080,
		Resources:     resources,
		RollingUpdate: rollout,
		Probes: &types.ProbesConfig{
			Readiness: &types.ProbeConfig{Path: "/healthz", Preset: "fast"},
		},
		Presets: reg,
	}
	svc := services.Spec{Name: "web", Port: 80, TargetPort: 8080, Selector: map[string]string{"app.kubernetes.io/name": "web"}}
	opts := types.Options{FieldManager: "shop-operator"}

	fmt.Println("\nVerbs:")
	run([]step{
		{"deployments.Create", func() (types.Result, error) { return deployments.Create(ctx, kube, app, web) }},
		{"deployments.Create", func() (types.Result, error) { return deployments.Create(ctx, kube, app, web) }},
		{"deployments.Apply", func() (types.Result, error) { return deployments.Apply(ctx, kube, app, web, opts) }},
		{"services.Create", func() (types.Result, error) { return services.Create(ctx, kube, app, svc) }},
	})

	d, err := kube.AppsV1().Deployments("team").Get(ctx, "web", metav1.GetOptions{})
	must(err)
	ref := d.OwnerReferences[0]
	probe := d.Spec.Template.Spec.Containers[0].ReadinessProbe
	fmt.Printf("\nThe Deployment in the cluster:\n")
	fmt.Printf("  owner      %s %s\n", ref.Kind, ref.Name)
	fmt.Printf("  resources  %s\n", d.Spec.Template.Spec.Containers[0].Resources.Requests.Cpu())
	fmt.Printf("  readiness  %s every %ds (the fast preset)\n", probe.HTTPGet.Path, probe.PeriodSeconds)

	// 3. Deletion that respects ownership.
	other := owner("billing", "team")
	deleteAs := func(o *unstructured.Unstructured) func() (types.Result, error) {
		return func() (types.Result, error) {
			return deployments.DeleteIfOwned(ctx, kube, "web", "team", types.OwnedByReference(o))
		}
	}
	fmt.Println("\nDeletion:")
	run([]step{
		{"DeleteIfOwned as billing", deleteAs(other)},
		{"DeleteIfOwned as shop", deleteAs(app)},
		{"DeleteIfOwned as shop", deleteAs(app)},
	})
}

// owner returns an App custom resource to own the objects.
func owner(name, namespace string) *unstructured.Unstructured {
	u := &unstructured.Unstructured{}
	u.SetAPIVersion("shop.example.com/v1")
	u.SetKind("App")
	u.SetName(name)
	u.SetNamespace(namespace)
	u.SetUID(k8stypes.UID(name + "-uid"))
	return u
}

// step is one call to make and print.
type step struct {
	call string
	do   func() (types.Result, error)
}

// run makes each call and prints its result.
func run(steps []step) {
	for _, s := range steps {
		res, err := s.do()
		must(err)
		fmt.Printf("  %-26s %s\n", s.call, res)
	}
}

func must(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
