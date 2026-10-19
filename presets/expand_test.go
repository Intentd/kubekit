package presets_test

import (
	"testing"

	"github.com/intentd/kubekit/core/types"
	"github.com/intentd/kubekit/presets"
)

func TestExpandResources(t *testing.T) {
	reg := presets.Builtin()

	got, err := presets.ExpandResources(&types.ResourceRequirements{Preset: "small"}, reg)
	if err != nil || got.Requests["cpu"] != "100m" {
		t.Errorf("small = %+v, %v; want the small preset's requests", got, err)
	}

	explicit := &types.ResourceRequirements{Requests: map[string]string{"cpu": "1"}}
	if got, _ := presets.ExpandResources(explicit, reg); got != explicit {
		t.Error("explicit requirements should be returned as they are")
	}
	if got, err := presets.ExpandResources(&types.ResourceRequirements{}, reg); got != nil || err != nil {
		t.Errorf("empty requirements = %+v, %v; want nil", got, err)
	}
	if got, err := presets.ExpandResources(nil, reg); got != nil || err != nil {
		t.Errorf("nil = %+v, %v; want nil", got, err)
	}
	if _, err := presets.ExpandResources(&types.ResourceRequirements{Preset: "huge"}, reg); err == nil {
		t.Error("an unknown preset should be an error")
	}
}

func TestExpandSecurity(t *testing.T) {
	reg := presets.Builtin()

	c, err := presets.ExpandContainerSecurity(&types.ContainerSecurityContext{Preset: "restricted"}, reg)
	if err != nil || !*c.RunAsNonRoot {
		t.Errorf("restricted container = %+v, %v; want runAsNonRoot", c, err)
	}
	p, err := presets.ExpandPodSecurity(&types.PodSecurityContext{Preset: "hardened"}, reg)
	if err != nil || *p.RunAsUser != 65534 {
		t.Errorf("hardened pod = %+v, %v; want runAsUser 65534", p, err)
	}
	if _, err := presets.ExpandPodSecurity(&types.PodSecurityContext{Preset: "unknown"}, reg); err == nil {
		t.Error("an unknown preset should be an error")
	}
}

func TestExpandRollingUpdate(t *testing.T) {
	reg := presets.Builtin()

	got, err := presets.ExpandRollingUpdate(&types.RollingUpdateBehavior{Preset: "safe"}, reg)
	if err != nil || got.MaxSurge != "1" || got.MaxUnavailable != "0" {
		t.Errorf("safe = %+v, %v; want 1/0", got, err)
	}
	explicit := &types.RollingUpdateBehavior{MaxSurge: "2"}
	if got, _ := presets.ExpandRollingUpdate(explicit, reg); got == explicit || got.MaxSurge != "2" {
		t.Errorf("explicit = %+v; want a copy of it", got)
	}
	if got, err := presets.ExpandRollingUpdate(nil, reg); got != nil || err != nil {
		t.Errorf("nil = %+v, %v; want nil", got, err)
	}
	if got, err := presets.ExpandRollingUpdate(&types.RollingUpdateBehavior{Preset: "slow"}, reg); got != nil || err == nil {
		t.Errorf("unknown = %+v, %v; want nil and an error", got, err)
	}
}

func TestExpandHPA(t *testing.T) {
	reg := presets.Builtin()

	b, target, err := presets.ExpandHPA(&types.HPABehavior{Preset: "web"}, 0, reg)
	if err != nil || b.ScaleUp == nil || target != 70 {
		t.Errorf("web = %+v, %d, %v; want its behavior and target 70", b, target, err)
	}
	if _, target, _ := presets.ExpandHPA(&types.HPABehavior{Preset: "web"}, 55, reg); target != 55 {
		t.Errorf("target = %d; a target already set should win", target)
	}
	explicit := &types.HPABehavior{ScaleDown: &types.HPAScalingRules{StabilizationWindowSeconds: 60}}
	if b, target, _ := presets.ExpandHPA(explicit, 80, reg); b == explicit || b.ScaleDown.StabilizationWindowSeconds != 60 || target != 80 {
		t.Errorf("explicit = %+v, %d; want a copy and target 80", b, target)
	}
	if b, target, err := presets.ExpandHPA(nil, 80, reg); b != nil || target != 80 || err != nil {
		t.Errorf("nil = %+v, %d, %v; want nil and target 80", b, target, err)
	}
	if b, target, err := presets.ExpandHPA(&types.HPABehavior{Preset: "nope"}, 0, reg); b != nil || target != 0 || err == nil {
		t.Errorf("unknown = %+v, %d, %v; want nil, 0 and an error", b, target, err)
	}
}

func TestExpandPDB(t *testing.T) {
	reg := presets.Builtin()

	if minA, maxU, err := presets.ExpandPDB("rolling", "", "", reg); err != nil || minA != "" || maxU != "1" {
		t.Errorf("rolling = %q/%q, %v; want maxUnavailable 1", minA, maxU, err)
	}
	if _, maxU, _ := presets.ExpandPDB("rolling", "", "2", reg); maxU != "2" {
		t.Errorf("maxUnavailable = %q; a value already set should win", maxU)
	}
	if minA, maxU, err := presets.ExpandPDB("", "1", "", reg); err != nil || minA != "1" || maxU != "" {
		t.Errorf("no preset = %q/%q, %v; want the values as given", minA, maxU, err)
	}
	if minA, _, err := presets.ExpandPDB("nope", "1", "", reg); err == nil || minA != "1" {
		t.Errorf("unknown = %q, %v; want the values as given and an error", minA, err)
	}
}

func TestExpandNetworkPolicy(t *testing.T) {
	reg := presets.Builtin()
	given := presets.NetworkPolicyRules{PolicyTypes: []string{"Ingress"}}

	got, err := presets.ExpandNetworkPolicy("deny-all", given, reg)
	if err != nil || len(got.PolicyTypes) != 2 || got.Ingress == nil || got.Egress == nil {
		t.Errorf("deny-all = %+v, %v; want empty ingress and egress, both policy types", got, err)
	}
	if got, err := presets.ExpandNetworkPolicy("", given, reg); err != nil || len(got.PolicyTypes) != 1 {
		t.Errorf("no preset = %+v, %v; want the rules as given", got, err)
	}
	if got, err := presets.ExpandNetworkPolicy("nope", given, reg); err == nil || len(got.PolicyTypes) != 1 {
		t.Errorf("unknown = %+v, %v; want the rules as given and an error", got, err)
	}
}

func TestExpandResourceQuota(t *testing.T) {
	reg := presets.Builtin()
	given := map[string]string{"pods": "3"}

	if got, err := presets.ExpandResourceQuota("small", given, reg); err != nil || got["pods"] != "10" {
		t.Errorf("small = %v, %v; want the preset's hard limits", got, err)
	}
	if got, _ := presets.ExpandResourceQuota("", given, reg); got["pods"] != "3" {
		t.Errorf("no preset = %v; want hard as given", got)
	}
	if got, err := presets.ExpandResourceQuota("nope", given, reg); err == nil || got["pods"] != "3" {
		t.Errorf("unknown = %v, %v; want hard as given and an error", got, err)
	}
}

func TestExpandLimitRange(t *testing.T) {
	item := types.LimitRangeItem{Type: "Container"}
	reg := presets.Registry{LimitRanges: []presets.LimitRange{{Name: "team", Limits: []types.LimitRangeItem{item, item}}}}
	given := []types.LimitRangeItem{item}

	if got, err := presets.ExpandLimitRange("team", nil, reg); err != nil || len(got) != 2 {
		t.Errorf("team = %v, %v; want the preset's two items", got, err)
	}
	if got, err := presets.ExpandLimitRange("team", given, reg); err != nil || len(got) != 1 {
		t.Errorf("limits set = %v, %v; want them to win over the preset", got, err)
	}
	if got, err := presets.ExpandLimitRange("nope", nil, reg); err == nil || got != nil {
		t.Errorf("unknown = %v, %v; want nil and an error", got, err)
	}
}
