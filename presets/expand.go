package presets

import (
	"fmt"

	"github.com/intentd/kubewright/core/types"
)

// ExpandResources resolves resource requirements: if resources.preset is set
// it expands to explicit requests/limits; otherwise the block is returned as-is.
// Returns nil when neither preset nor explicit values are declared, and an
// error for an unknown preset.
func ExpandResources(r *types.ResourceRequirements, reg Registry) (*types.ResourceRequirements, error) {
	if r == nil {
		return nil, nil
	}
	if r.Preset != "" {
		p, ok := reg.Resource(r.Preset)
		if !ok {
			return nil, fmt.Errorf("unknown resources preset %q", r.Preset)
		}
		return p.Requirements(), nil
	}
	if len(r.Requests) == 0 && len(r.Limits) == 0 {
		return nil, nil
	}
	return r, nil
}

// ExpandContainerSecurity resolves a ContainerSecurityContext:
// if a preset is set it expands to explicit fields; otherwise the block is
// returned as-is; an unknown preset is an error. Returns nil when sc is nil.
func ExpandContainerSecurity(sc *types.ContainerSecurityContext, reg Registry) (*types.ContainerSecurityContext, error) {
	if sc == nil {
		return nil, nil
	}
	if sc.Preset != "" {
		p, ok := reg.ContainerSecurity(sc.Preset)
		if !ok {
			return nil, fmt.Errorf("unknown securityContext preset %q", sc.Preset)
		}
		return p.Context(), nil
	}
	return sc, nil
}

// ExpandPodSecurity resolves a PodSecurityContext:
// if a preset is set it expands to explicit fields; otherwise the block is
// returned as-is; an unknown preset is an error. Returns nil when ps is nil.
func ExpandPodSecurity(ps *types.PodSecurityContext, reg Registry) (*types.PodSecurityContext, error) {
	if ps == nil {
		return nil, nil
	}
	if ps.Preset != "" {
		p, ok := reg.PodSecurity(ps.Preset)
		if !ok {
			return nil, fmt.Errorf("unknown podSecurity preset %q", ps.Preset)
		}
		return p.Context(), nil
	}
	return ps, nil
}

// ExpandRollingUpdate resolves a RollingUpdateBehavior: if a preset is set it
// expands to MaxSurge and MaxUnavailable; otherwise a copy of the block is
// returned. Returns nil when r is nil, and nil with an error for an unknown
// preset.
func ExpandRollingUpdate(r *types.RollingUpdateBehavior, reg Registry) (*types.RollingUpdateBehavior, error) {
	if r == nil {
		return nil, nil
	}
	if r.Preset != "" {
		p, ok := reg.RollingUpdate(r.Preset)
		if !ok {
			return nil, fmt.Errorf("unknown rollingUpdate preset %q", r.Preset)
		}
		return p.Behavior(), nil
	}
	out := *r
	return &out, nil
}

// ExpandHPA resolves an HPA's scaling behavior and CPU target. If b names a
// preset, the preset's behavior replaces it, and its CPU target fills
// cpuTarget when that is 0. Otherwise a copy of b and cpuTarget are returned.
// For an unknown preset it returns a nil behavior, cpuTarget and an error.
func ExpandHPA(b *types.HPABehavior, cpuTarget int32, reg Registry) (*types.HPABehavior, int32, error) {
	if b == nil {
		return nil, cpuTarget, nil
	}
	if b.Preset != "" {
		p, ok := reg.HPA(b.Preset)
		if !ok {
			return nil, cpuTarget, fmt.Errorf("unknown hpa preset %q", b.Preset)
		}
		out := types.HPABehavior{}
		if p.Behavior != nil {
			out = *p.Behavior
		}
		if cpuTarget == 0 {
			cpuTarget = p.CPUTarget()
		}
		return &out, cpuTarget, nil
	}
	out := *b
	return &out, cpuTarget, nil
}

// ExpandPDB fills a PodDisruptionBudget's minAvailable and maxUnavailable from
// the preset called name, keeping any value already set. An empty name
// returns them as they are; an unknown one returns them with an error.
func ExpandPDB(name, minAvailable, maxUnavailable string, reg Registry) (string, string, error) {
	if name == "" {
		return minAvailable, maxUnavailable, nil
	}
	p, ok := reg.PDB(name)
	if !ok {
		return minAvailable, maxUnavailable, fmt.Errorf("unknown pdb preset %q", name)
	}
	if minAvailable == "" {
		minAvailable = p.MinAvailable
	}
	if maxUnavailable == "" {
		maxUnavailable = p.MaxUnavailable
	}
	return minAvailable, maxUnavailable, nil
}

// NetworkPolicyRules is what a network policy preset sets.
type NetworkPolicyRules struct {
	Ingress     []types.NetworkPolicyIngressRule
	Egress      []types.NetworkPolicyEgressRule
	PolicyTypes []string
}

// ExpandNetworkPolicy replaces rules with the preset called name: its ingress,
// egress and policy types. An empty name returns rules as they are; an
// unknown one returns them with an error.
func ExpandNetworkPolicy(name string, rules NetworkPolicyRules, reg Registry) (NetworkPolicyRules, error) {
	if name == "" {
		return rules, nil
	}
	p, ok := reg.NetworkPolicy(name)
	if !ok {
		return rules, fmt.Errorf("unknown networkPolicy preset %q", name)
	}
	return NetworkPolicyRules{Ingress: p.Ingress, Egress: p.Egress, PolicyTypes: p.PolicyTypes}, nil
}

// ExpandResourceQuota replaces hard with the preset called name. An empty
// name returns hard as it is; an unknown one returns it with an error.
func ExpandResourceQuota(name string, hard map[string]string, reg Registry) (map[string]string, error) {
	if name == "" {
		return hard, nil
	}
	p, ok := reg.ResourceQuota(name)
	if !ok {
		return hard, fmt.Errorf("unknown resourceQuota preset %q", name)
	}
	return p.Hard, nil
}

// ExpandLimitRange fills limits from the preset called name when limits is
// empty; limits that are set win over the preset. An empty name returns
// limits as they are; an unknown one returns them with an error.
func ExpandLimitRange(name string, limits []types.LimitRangeItem, reg Registry) ([]types.LimitRangeItem, error) {
	if name == "" || len(limits) > 0 {
		return limits, nil
	}
	p, ok := reg.LimitRange(name)
	if !ok {
		return limits, fmt.Errorf("unknown limitRange preset %q", name)
	}
	return p.Limits, nil
}
