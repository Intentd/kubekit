<p align="center">
  <img src="./assets/kuberesources.svg" alt="kuberesources, Kubernetes resources for Go" width="500">
</p>

Build and write Kubernetes resources for an owner from plain Go specs, with idempotent verbs and named presets for the settings every team repeats.

```go
res, err := deployments.Apply(ctx, clientset, app, deployments.Spec{
	Name:     "web",
	Image:    "nginx:1.27",
	Replicas: 2,
	Port:     8080,
	Probes: &types.ProbesConfig{
		Readiness: &types.ProbeConfig{Path: "/healthz", Preset: "fast"},
	},
	Presets: presets.Builtin(),
}, types.Options{FieldManager: "my-operator"})
// res == types.Applied; the Deployment is owned by app and goes when app goes
```

kuberesources is a library, not a framework. There is no controller, no reconcile loop and no logging. It turns a spec into an object, sets its owner reference, writes it with the call you choose, and tells you what happened. Every verb is idempotent: call it again and you get `Unchanged` or `NotFound`, not an error. Your controller, CLI or test decides when to call it.

## Why

Every operator writes the same code for its child resources. Build a Deployment from a few fields. Set the owner reference so it is garbage collected. Create it if it is missing, and don't fail if it is already there. Write it with server-side apply. Delete it, but only if this owner made it. Copy a Secret into the namespaces that need it.

None of that is specific to one operator, and getting it right takes care: a selector that matches the pod template, a field manager on every apply, a not-found that is an outcome rather than an error. kuberesources is that code, written once for each kind, with the same verbs and results for each.

Every team also repeats the same settings: requests and limits sized small, medium or large, probe timings for a fast API or a slow-starting JVM, a restricted security context, a rollout that never drops capacity. Presets give those settings names. A spec says `medium`, `restricted` or `safe`, and the preset fills in the fields. The built-ins cover the common cases, and your own presets add to them or replace them.

## Install

```bash
go get github.com/intentd/kuberesources
```

## The verbs

Each kind is a package under `core`, and each package has the same verbs:

| Verb | Does | Results |
|---|---|---|
| `Build(owner, spec)` | returns the object, with the owner reference set. No I/O. | |
| `Create(ctx, kube, owner, spec)` | creates the object if it does not exist | `Created`, `Unchanged` |
| `Apply(ctx, kube, owner, spec, opts)` | writes it with server-side apply | `Applied`, `Unchanged`, `Replaced` |
| `Delete(ctx, kube, owner, spec)` | deletes it | `Deleted`, `NotFound` |
| `DeleteIfOwned(ctx, kube, name, namespace, owned)` | deletes it only if `owned` says it is yours | `Deleted`, `NotFound`, `NotOwned` |

`kube` is a `kubernetes.Interface`, so a real clientset and `fake.NewClientset()` both work. The owner is any object with metadata, such as your custom resource, a typed object or an `unstructured.Unstructured`. A spec with no namespace uses the owner's.

Every expected outcome comes back as a `types.Result` with a nil error. Calling a verb twice is safe: an object that already exists is `Unchanged`, and one already gone is `NotFound`. An error means the call failed, such as an invalid spec or a refused request. The library does not log. Log the result if you want to:

```go
res, err := services.Create(ctx, clientset, app, spec)
if err != nil {
	return err
}
log.Printf("service %s: %s", spec.Name, res) // "service web: created"
```

## Kinds

Each built-in kind is a package under [`core`](core), named for the kind in plural: `core/deployments`, `core/secrets`, `core/clusterroles`. Where the API differs, a package does too, such as no `Apply` for a kind that can't change after it is created. Each package's doc says how. `customresources` covers any other kind through a `dynamic.Interface` and a `meta.RESTMapper`.

## Owners

`Build` sets one controller owner reference to the owner, so Kubernetes deletes the object when the owner is deleted. A cluster-scoped kind (`namespaces`, `pvs`, `clusterroles`, `clusterrolebindings`) gets the reference only from a cluster-scoped owner, because Kubernetes doesn't let a namespaced object own one. For those, use `DeleteIfOwned` with a predicate of your own, such as one that checks a label.

`DeleteIfOwned` takes the predicate rather than an owner, so you decide what ownership means. `types.OwnedByReference(owner)` checks the owner reference:

```go
res, err := configmaps.DeleteIfOwned(ctx, clientset, "web-config", "team", types.OwnedByReference(app))
// NotOwned if another owner made it: it is left alone
```

## Server-side apply

Every `Apply` writes through one function, `apply.Typed`, so all kinds apply the same way. `types.Options` sets the field manager and whether to take fields another manager owns:

```go
opts := types.Options{FieldManager: "my-operator", Force: true}
```

An empty field manager means `kuberesources`. Without `Force`, a field another manager owns is a conflict, returned as an error.

`apply.Object` writes any object, typed or unstructured, through the dynamic client. Use it for objects you build yourself:

```go
d := deployments.Build(app, spec)
d.TypeMeta = metav1.TypeMeta{APIVersion: "apps/v1", Kind: "Deployment"}
res, err := apply.Object(ctx, dyn, mapper, d, opts)
```

## Presets

Some fields are the same settings repeated: resource requests and limits, probe timings, security contexts. A field can name a preset instead of spelling them out:

| Presets for | Built in |
|---|---|
| Resources | `tiny`, `small`, `medium`, `large`, `burst`, `steady`, `compute-heavy`, `memory-heavy` |
| Probes | `fast`, `standard`, `patient`, `slow-start` |
| Container security | `baseline`, `restricted`, `hardened` |
| Pod security | `baseline`, `restricted`, `hardened` |
| Rolling update | `safe`, `fast`, `blue-green` |
| HPA | `web`, `api`, `latency-sensitive`, `batch`, `cost-optimized` |
| PDB | `zero-downtime`, `rolling`, `relaxed` |
| Network policy | `deny-all`, `deny-all-ingress`, `deny-all-egress`, `allow-same-namespace`, `allow-dns-egress` |
| Resource quota | `small`, `medium`, `large`, `xlarge` |

`presets.Builtin()` returns them as data. Add your own, or replace a built-in, with `With`. Names ignore case:

```go
reg := presets.Builtin().With(presets.Registry{
	Resources: []presets.Resource{{
		Name:     "myorg-gpu",
		Requests: map[string]string{"cpu": "4", "memory": "16Gi", "nvidia.com/gpu": "1"},
	}},
})

r, err := presets.ExpandResources(&types.ResourceRequirements{Preset: "myorg-gpu"}, reg)
```

Expand a preset before you build: `ExpandResources`, `ExpandContainerSecurity` and `ExpandPodSecurity` turn a named preset into explicit fields, and return an error for an unknown one, so a typo doesn't quietly give you no limits. Probe presets are the exception: the workload kinds look them up when they build, in the spec's `Presets` registry (pass `presets.Builtin()` or your merged one), and a name they don't find falls back to `standard`.

## Testing

Because every verb takes a `kubernetes.Interface`, `fake.NewClientset()` from client-go runs them in a unit test, server-side apply included:

```go
cs := fake.NewClientset()
res, _ := deployments.Create(ctx, cs, app, spec) // Created
res, _ = deployments.Create(ctx, cs, app, spec)  // Unchanged
```

## Status

kuberesources is before 1.0. Names and signatures may still change between minor versions, and release notes list every breaking change. It started inside [Inrun](https://github.com/intentd/inrun), which builds every child resource with it, and is now a separate module with no dependency on Inrun.

## Contributing

A good change:

- keeps the verbs the same across kinds. A new kind has `Build`, `Create`, `Apply`, `Delete` and `DeleteIfOwned` unless the API rules one out
- keeps `Build` pure and every expected outcome a `Result`, not an error
- writes through `apply.Typed` or `apply.Object`, not its own patch
- adds a test with `TestBuild` and `TestLifecycle`, as the existing kinds do

`go test ./...` runs every kind against the fake clientset.

## License

[Apache 2.0](LICENSE)
