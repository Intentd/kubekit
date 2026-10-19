<div align="center">
  <img src="./assets/kubewright.svg" alt="kubewright, Kubernetes building blocks for Go" height="64" />

  <p><strong>The child-resource code every operator writes, written once.</strong></p>

  <p>
    <a href="https://github.com/intentd/kubewright/actions/workflows/ci.yml"><img src="https://github.com/intentd/kubewright/actions/workflows/ci.yml/badge.svg" alt="CI" /></a>
    <a href="https://github.com/intentd/kubewright/tags"><img src="https://img.shields.io/github/v/tag/intentd/kubewright?label=version" alt="Version" /></a>
    <a href="https://pkg.go.dev/github.com/intentd/kubewright"><img src="https://pkg.go.dev/badge/github.com/intentd/kubewright.svg" alt="Go Reference" /></a>
    <img src="https://img.shields.io/badge/Go-1.27+-00ADD8.svg" alt="Go 1.27+" />
    <img src="https://img.shields.io/badge/Kubernetes-1.37-326CE5.svg" alt="Kubernetes 1.37" />
    <img src="https://img.shields.io/badge/license-Apache%202.0-blue.svg" alt="License" />
  </p>
</div>

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

Build and write Kubernetes **resources** for an owner from plain Go specs, with idempotent verbs and named **presets** for the settings every team repeats.

kubewright is a library, not a framework. There is no controller, no reconcile loop and no logging. It turns a spec into an object, sets its owner reference, writes it with the call you choose, and tells you what happened. Every verb is idempotent: call it again and you get `Unchanged` or `NotFound`, not an error. Your controller, CLI or test decides when to call it.

## Why

Every operator writes the same code for its child resources. Build a Deployment from a few fields. Set the owner reference so it is garbage collected. Create it if it is missing, and don't fail if it is already there. Write it with server-side apply. Delete it, but only if this owner made it. Copy a Secret into the namespaces that need it.

None of that is specific to one operator, and getting it right takes care: a selector that matches the pod template, a field manager on every apply, a not-found that is an outcome rather than an error. And the Kubernetes API hardly moves: a Deployment, a Service and an owner reference have the same shape in every cluster, and server-side apply works the same everywhere. So that code can be written once, tested once and used everywhere: in a controller, a CLI or a test. kubewright is that code, with the same verbs and results for each kind.

Every team also repeats the same settings: requests and limits sized small, medium or large, probe timings for a fast API or a slow-starting JVM, a restricted security context, a rollout that never drops capacity. **Presets** give those settings names. You write `medium`, `restricted` or `safe`, and expanding the preset fills in the fields. The built-ins cover the common cases, and your own presets add to them or replace them.

## Install

```bash
go get github.com/intentd/kubewright
```

A complete program is in [examples/main.go](examples/main.go): an App that owns a Deployment and a Service, with presets expanded, each verb's result printed, and a delete that leaves another owner's objects alone. It runs against client-go's fake clientset, so it needs no cluster: `go run ./examples`.

## The verbs

Each kind is a package under `core`, and each package has the same verbs:

| Verb | Does | Results |
|---|---|---|
| `Build(owner, spec)` | returns the object, with the owner reference set. No I/O. | |
| `Create(ctx, kube, owner, spec)` | creates the object if it does not exist | `Created`, `Unchanged` |
| `Apply(ctx, kube, owner, spec, opts)` | writes it with server-side apply | `Applied`, `Unchanged`, `Replaced` |
| `Delete(ctx, kube, owner, spec)` | deletes it | `Deleted`, `NotFound` |
| `DeleteIfOwned(ctx, kube, name, namespace, owned)` | deletes it only if `owned` says it is yours | `Deleted`, `NotFound`, `NotOwned` |

For a cluster-scoped kind, `DeleteIfOwned` takes no namespace: `DeleteIfOwned(ctx, kube, name, owned)`.

`Create` makes an object that is missing and never changes one that exists: an existing object is `Unchanged` even when it differs from the spec. To bring an existing object in line with the spec, use `Apply`.

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

Each built-in kind is a package under [`core`](core), named for the kind in plural: `core/deployments`, `core/secrets`, `core/clusterroles`.

Every kind has all five verbs, except these:

| Kind | Differs |
|---|---|
| `serviceaccounts` | no `Apply`: nothing in a ServiceAccount drifts after it is created |
| `jobs` | no `Apply`: a Job's pod template can't change after it is created. No `DeleteIfOwned` |
| `customresources` | no `Delete`; delete with `DeleteIfOwned` |

`customresources` covers any other kind, your own custom resources included, through a `dynamic.Interface` and a `meta.RESTMapper`.

A few kinds have one more function, for a job their verbs don't cover:

- `secrets.CopyToNamespaces` copies a Secret into each of a list of namespaces, each copy owned by the owner. A namespace that already has it is skipped.
- `jobs.CreateForDelete` creates a Job with no owner reference, for cleanup work that runs while the owner is being deleted and must outlive it.

```go
err := secrets.CopyToNamespaces(ctx, clientset, app, secrets.Spec{
	Name:          "db-credentials",
	FromSecret:    "master-db-creds",
	FromNamespace: "platform",
}, []string{"team", "monitoring", "staging"})
```

## Owners

`Build` sets one controller owner reference to the owner, so Kubernetes deletes the object when the owner is deleted. A cluster-scoped kind (`namespaces`, `pvs`, `clusterroles`, `clusterrolebindings`) gets the reference only from a cluster-scoped owner, because Kubernetes doesn't let a namespaced object own one.

`DeleteIfOwned` takes a predicate rather than an owner, so you decide what ownership means. Three come with the library:

| Predicate | Owned when the object | Use it for |
|---|---|---|
| `types.OwnedByReference(owner)` | has an owner reference to `owner` | anything `Build` set a reference on |
| `types.OwnedByLabels(labels)` | carries every one of these labels | objects that can't hold a reference, such as a cluster-scoped object with a namespaced owner |
| `types.OwnedByAnnotations(annotations)` | carries every one of these annotations | the same, when the marker shouldn't be selectable or is too long for a label |

An empty set of labels or annotations matches nothing, so a missing value never deletes by accident.

```go
res, err := configmaps.DeleteIfOwned(ctx, clientset, "web-config", "team-namespace", types.OwnedByReference(app))
// NotOwned if another owner made it: it is left alone
```

For a cluster-scoped object with a namespaced owner, mark it when you create it, and delete by the same mark:

```go
mine := map[string]string{"app.example.com/owner": "team.shop"}

clusterroles.Create(ctx, clientset, app, clusterroles.Spec{Name: "shop-reader", Labels: mine, Rules: rules})
res, err := clusterroles.DeleteIfOwned(ctx, clientset, "shop-reader", types.OwnedByLabels(mine))
```

## Server-side apply

Every `Apply` writes through one function, `apply.Typed`, so all kinds apply the same way. `types.Options` sets the field manager and whether to take fields another manager owns:

```go
opts := types.Options{FieldManager: "my-operator", Force: true}
```

An empty field manager means `kubewright`. Without `Force`, a field another manager owns is a conflict, returned as an error.

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

Expand a preset into explicit fields before you build. There is one `Expand` function per kind of preset, and each returns an error for an unknown name, so a typo doesn't quietly give you no limits:

```go
spec.RollingUpdate, err = presets.ExpandRollingUpdate(&types.RollingUpdateBehavior{Preset: "safe"}, reg)
spec.Hard, err = presets.ExpandResourceQuota("small", nil, reg)
```

Each one keeps the rule that fits its field. A PDB or limit range preset fills only what you left empty. A network policy or resource quota preset replaces the rules. An HPA preset sets the scaling behavior, and the CPU target too when you haven't set one.

Probe presets are the exception: the workload kinds look them up when they build, in the spec's `Presets` registry (pass `presets.Builtin()` or your merged one), and a name they don't find falls back to `standard`.

## Testing

Because every verb takes a `kubernetes.Interface`, `fake.NewClientset()` from client-go runs them in a unit test, server-side apply included:

```go
cs := fake.NewClientset()
res, _ := deployments.Create(ctx, cs, app, spec) // Created
res, _ = deployments.Create(ctx, cs, app, spec)  // Unchanged
```

## Reading what you built

kubewright writes objects and tells you what the write did. To ask how they are doing afterwards, such as whether a rollout finished or a pod is crash looping, see [KEL](https://github.com/intentd/kel), its companion library for reading Kubernetes state.

## Status

kubewright is before 1.0. Names and signatures may still change between minor versions. It started inside [Inrun](https://github.com/intentd/inrun), which builds every child resource with it, and is now a separate module with no dependency on Inrun.

## Contributing

A good change:

- keeps the verbs the same across kinds. A new kind has `Build`, `Create`, `Apply`, `Delete` and `DeleteIfOwned` unless the API rules one out
- keeps `Build` pure and every expected outcome a `Result`, not an error
- writes through `apply.Typed` or `apply.Object`, not its own patch
- adds a test with `TestBuild` and `TestLifecycle`, as the existing kinds do

`go test ./...` runs every kind against the fake clientset.

## License

[Apache 2.0](LICENSE)
