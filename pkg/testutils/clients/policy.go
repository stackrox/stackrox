package clients

import (
	"context"
	"fmt"
	"time"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
	"google.golang.org/grpc"
)

type policyClient struct {
	conn    *grpc.ClientConn
	service v1.PolicyServiceClient
}

func (p *policyClient) CreatePolicy(ctx context.Context, config *PolicyConfig) (*storage.Policy, error) {
	if p.service == nil {
		p.service = v1.NewPolicyServiceClient(p.conn)
	}

	policy := p.buildPolicyFromConfig(config)

	resp, err := p.service.PostPolicy(ctx, &v1.PostPolicyRequest{
		Policy:                 policy,
		EnableStrictValidation: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create policy: %w", err)
	}

	return resp.Policy, nil
}

func (p *policyClient) GetPolicy(ctx context.Context, policyID string) (*storage.Policy, error) {
	if p.service == nil {
		p.service = v1.NewPolicyServiceClient(p.conn)
	}

	resp, err := p.service.GetPolicy(ctx, &v1.ResourceByID{Id: policyID})
	if err != nil {
		return nil, fmt.Errorf("failed to get policy %s: %w", policyID, err)
	}

	return resp, nil
}

func (p *policyClient) UpdatePolicy(ctx context.Context, policy *storage.Policy) (*storage.Policy, error) {
	if p.service == nil {
		p.service = v1.NewPolicyServiceClient(p.conn)
	}

	resp, err := p.service.PutPolicy(ctx, &v1.PutPolicyRequest{
		Policy: policy,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update policy %s: %w", policy.GetId(), err)
	}

	return resp, nil
}

func (p *policyClient) DeletePolicy(ctx context.Context, policyID string) error {
	if p.service == nil {
		p.service = v1.NewPolicyServiceClient(p.conn)
	}

	_, err := p.service.DeletePolicy(ctx, &v1.ResourceByID{Id: policyID})
	if err != nil {
		return fmt.Errorf("failed to delete policy %s: %w", policyID, err)
	}

	return nil
}

func (p *policyClient) ListPolicies(ctx context.Context) ([]*storage.ListPolicy, error) {
	if p.service == nil {
		p.service = v1.NewPolicyServiceClient(p.conn)
	}

	resp, err := p.service.ListPolicies(ctx, &v1.RawQuery{})
	if err != nil {
		return nil, fmt.Errorf("failed to list policies: %w", err)
	}

	return resp.GetPolicies(), nil
}

func (p *policyClient) ImportPolicies(ctx context.Context, policies []*storage.Policy) error {
	if p.service == nil {
		p.service = v1.NewPolicyServiceClient(p.conn)
	}

	_, err := p.service.ImportPolicies(ctx, &v1.ImportPoliciesRequest{
		Policies: policies,
		Metadata: &v1.ImportPoliciesMetadata{
			Overwrite: true,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to import policies: %w", err)
	}

	return nil
}

// buildPolicyFromConfig converts a PolicyConfig to a storage.Policy
func (p *policyClient) buildPolicyFromConfig(config *PolicyConfig) *storage.Policy {
	policy := &storage.Policy{
		Name:        config.Name,
		Description: config.Description,
		Categories:  config.Categories,
		Disabled:    false,
		Severity:    config.Severity,
		EnforcementActions: p.buildEnforcementActions(config.Enforcement),
		PolicySections:     p.buildPolicySections(config),
		LifecycleStages:    p.buildLifecycleStages(config.Scope),
		EventSource:        storage.EventSource_DEPLOYMENT_EVENT,
		Scope:              p.buildPolicyScope(config),
	}

	// Set default severity if not specified
	if policy.Severity == storage.Severity_UNSET_SEVERITY {
		policy.Severity = storage.Severity_HIGH_SEVERITY
	}

	return policy
}

func (p *policyClient) buildEnforcementActions(enforce bool) []storage.EnforcementAction {
	if enforce {
		return []storage.EnforcementAction{
			storage.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT,
			storage.EnforcementAction_UNSATISFIABLE_NODE_CONSTRAINT_ENFORCEMENT,
		}
	}
	return []storage.EnforcementAction{
		// Monitor mode - no enforcement actions
	}
}

func (p *policyClient) buildLifecycleStages(scope PolicyScope) []storage.LifecycleStage {
	switch scope {
	case RuntimeScope:
		return []storage.LifecycleStage{
			storage.LifecycleStage_RUNTIME,
		}
	case BuildScope:
		return []storage.LifecycleStage{
			storage.LifecycleStage_BUILD,
		}
	default:
		return []storage.LifecycleStage{
			storage.LifecycleStage_DEPLOY,
			storage.LifecycleStage_RUNTIME,
		}
	}
}

func (p *policyClient) buildPolicyScope(config *PolicyConfig) []*storage.Scope {
	// Default scope includes all namespaces
	return []*storage.Scope{
		{
			Label: &storage.Scope_Namespace{
				Namespace: "*",
			},
		},
	}
}

func (p *policyClient) buildPolicySections(config *PolicyConfig) []*storage.PolicySection {
	sections := []*storage.PolicySection{}

	// Runtime policy configurations
	if config.RuntimeConfig != nil {
		sections = append(sections, p.buildRuntimeSections(config.RuntimeConfig)...)
	}

	// Build policy configurations
	if config.BuildConfig != nil {
		sections = append(sections, p.buildBuildSections(config.BuildConfig)...)
	}

	// Network policy configurations
	if config.NetworkPolicyConfig != nil {
		sections = append(sections, p.buildNetworkSections(config.NetworkPolicyConfig)...)
	}

	// If no specific config provided, create a basic policy section based on categories
	if len(sections) == 0 {
		sections = p.buildDefaultSections(config.Categories)
	}

	return sections
}

func (p *policyClient) buildRuntimeSections(config *RuntimePolicyConfig) []*storage.PolicySection {
	sections := []*storage.PolicySection{}

	if config.RequireNonRoot {
		sections = append(sections, &storage.PolicySection{
			SectionName: "Container Configuration",
			PolicyGroups: []*storage.PolicyGroup{
				{
					FieldName: "Privileged Container",
					BooleanOperator: storage.BooleanOperator_OR,
					Negate:    false,
					Values: []*storage.PolicyValue{
						{Value: "false"},
					},
				},
			},
		})
	}

	if config.BlockPrivileged {
		sections = append(sections, &storage.PolicySection{
			SectionName: "Privileged Container",
			PolicyGroups: []*storage.PolicyGroup{
				{
					FieldName: "Privileged Container",
					BooleanOperator: storage.BooleanOperator_OR,
					Negate:    true,
					Values: []*storage.PolicyValue{
						{Value: "true"},
					},
				},
			},
		})
	}

	if config.ReadOnlyRootFS {
		sections = append(sections, &storage.PolicySection{
			SectionName: "Read-Only Root Filesystem",
			PolicyGroups: []*storage.PolicyGroup{
				{
					FieldName: "Read-Only Root Filesystem",
					BooleanOperator: storage.BooleanOperator_OR,
					Negate:    false,
					Values: []*storage.PolicyValue{
						{Value: "true"},
					},
				},
			},
		})
	}

	if len(config.DisallowedCommands) > 0 {
		values := make([]*storage.PolicyValue, len(config.DisallowedCommands))
		for i, cmd := range config.DisallowedCommands {
			values[i] = &storage.PolicyValue{Value: cmd}
		}

		sections = append(sections, &storage.PolicySection{
			SectionName: "Disallowed Commands",
			PolicyGroups: []*storage.PolicyGroup{
				{
					FieldName: "Dockerfile Instruction Keyword",
					BooleanOperator: storage.BooleanOperator_OR,
					Negate:    true,
					Values:    values,
				},
			},
		})
	}

	return sections
}

func (p *policyClient) buildBuildSections(config *BuildPolicyConfig) []*storage.PolicySection {
	sections := []*storage.PolicySection{}

	if len(config.RequiredLabels) > 0 {
		for key, value := range config.RequiredLabels {
			sections = append(sections, &storage.PolicySection{
				SectionName: fmt.Sprintf("Required Label: %s", key),
				PolicyGroups: []*storage.PolicyGroup{
					{
						FieldName: "Required Label",
						BooleanOperator: storage.BooleanOperator_OR,
						Negate:    false,
						Values: []*storage.PolicyValue{
							{Value: fmt.Sprintf("%s=%s", key, value)},
						},
					},
				},
			})
		}
	}

	if config.MaxCVSSScore > 0 {
		sections = append(sections, &storage.PolicySection{
			SectionName: "Image Vulnerabilities",
			PolicyGroups: []*storage.PolicyGroup{
				{
					FieldName: "CVSS",
					BooleanOperator: storage.BooleanOperator_OR,
					Negate:    true,
					Values: []*storage.PolicyValue{
						{Value: fmt.Sprintf(">= %.1f", config.MaxCVSSScore)},
					},
				},
			},
		})
	}

	if len(config.BlockedRegistries) > 0 {
		values := make([]*storage.PolicyValue, len(config.BlockedRegistries))
		for i, registry := range config.BlockedRegistries {
			values[i] = &storage.PolicyValue{Value: registry}
		}

		sections = append(sections, &storage.PolicySection{
			SectionName: "Blocked Registries",
			PolicyGroups: []*storage.PolicyGroup{
				{
					FieldName: "Image Registry",
					BooleanOperator: storage.BooleanOperator_OR,
					Negate:    true,
					Values:    values,
				},
			},
		})
	}

	return sections
}

func (p *policyClient) buildNetworkSections(config *NetworkPolicyConfig) []*storage.PolicySection {
	sections := []*storage.PolicySection{}

	if config.BlockExternalEgress {
		sections = append(sections, &storage.PolicySection{
			SectionName: "Network Policy",
			PolicyGroups: []*storage.PolicyGroup{
				{
					FieldName: "Unexpected Network Flow Detected",
					BooleanOperator: storage.BooleanOperator_OR,
					Negate:    false,
					Values: []*storage.PolicyValue{
						{Value: "true"},
					},
				},
			},
		})
	}

	return sections
}

func (p *policyClient) buildDefaultSections(categories []string) []*storage.PolicySection {
	// Create default policy sections based on categories
	sections := []*storage.PolicySection{}

	for _, category := range categories {
		switch category {
		case "Privilege Escalation":
			sections = append(sections, &storage.PolicySection{
				SectionName: "Container Configuration",
				PolicyGroups: []*storage.PolicyGroup{
					{
						FieldName: "Privileged Container",
						BooleanOperator: storage.BooleanOperator_OR,
						Negate:    true,
						Values: []*storage.PolicyValue{
							{Value: "true"},
						},
					},
				},
			})
		case "Container Security":
			sections = append(sections, &storage.PolicySection{
				SectionName: "Container Security",
				PolicyGroups: []*storage.PolicyGroup{
					{
						FieldName: "Read-Only Root Filesystem",
						BooleanOperator: storage.BooleanOperator_OR,
						Negate:    false,
						Values: []*storage.PolicyValue{
							{Value: "true"},
						},
					},
				},
			})
		case "Image Vulnerabilities":
			sections = append(sections, &storage.PolicySection{
				SectionName: "Image Vulnerabilities",
				PolicyGroups: []*storage.PolicyGroup{
					{
						FieldName: "CVSS",
						BooleanOperator: storage.BooleanOperator_OR,
						Negate:    true,
						Values: []*storage.PolicyValue{
							{Value: ">= 7.0"},
						},
					},
				},
			})
		default:
			// Generic policy section for unknown categories
			sections = append(sections, &storage.PolicySection{
				SectionName: category,
				PolicyGroups: []*storage.PolicyGroup{
					{
						FieldName: "Container Name",
						BooleanOperator: storage.BooleanOperator_OR,
						Negate:    false,
						Values: []*storage.PolicyValue{
							{Value: ".*"},
						},
					},
				},
			})
		}
	}

	return sections
}