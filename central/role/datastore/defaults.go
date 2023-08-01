package datastore

import (
	"github.com/pkg/errors"
	rolePkg "github.com/stackrox/rox/central/role"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	permissionsUtils "github.com/stackrox/rox/pkg/auth/permissions/utils"
	"github.com/stackrox/rox/pkg/defaults/accesscontrol"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/utils"
)

type permSetAttributes struct {
	idSuffix           string
	postgresID         string // postgresID should be populated with valid UUID values.
	description        string
	resourceWithAccess []permissions.ResourceWithAccess
}

func (attributes *permSetAttributes) getID() string {
	return attributes.postgresID
}

var defaultPermissionSets = map[string]permSetAttributes{
	accesscontrol.Admin: {
		idSuffix:           "admin",
		postgresID:         accesscontrol.DefaultPermissionSetIDs[accesscontrol.Admin],
		description:        "For users: use it to provide read and write access to all the resources",
		resourceWithAccess: resources.AllResourcesModifyPermissions(),
	},
	accesscontrol.Analyst: {
		idSuffix:           "analyst",
		postgresID:         accesscontrol.DefaultPermissionSetIDs[accesscontrol.Analyst],
		resourceWithAccess: rolePkg.GetAnalystPermissions(),
		description:        "For users: use it to give read-only access to all the resources",
	},
	accesscontrol.ContinuousIntegration: {
		idSuffix:    "continuousintegration",
		postgresID:  accesscontrol.DefaultPermissionSetIDs[accesscontrol.ContinuousIntegration],
		description: "For automation: it includes the permissions required to enforce deployment policies",
		resourceWithAccess: []permissions.ResourceWithAccess{
			permissions.View(resources.Detection),
			permissions.Modify(resources.Image),
		},
	},
	accesscontrol.NetworkGraphViewer: {
		idSuffix:    "networkgraphviewer",
		postgresID:  accesscontrol.DefaultPermissionSetIDs[accesscontrol.NetworkGraphViewer],
		description: "For users: use it to give read-only access to the NetworkGraph pages",
		resourceWithAccess: []permissions.ResourceWithAccess{
			permissions.View(resources.Deployment),
			permissions.View(resources.NetworkGraph),
			permissions.View(resources.NetworkPolicy),
		},
	},
	accesscontrol.None: {
		idSuffix:    "none",
		postgresID:  accesscontrol.DefaultPermissionSetIDs[accesscontrol.None],
		description: "For users: use it to provide no read and write access to any resource",
	},
	accesscontrol.SensorCreator: {
		idSuffix:    "sensorcreator",
		postgresID:  accesscontrol.DefaultPermissionSetIDs[accesscontrol.SensorCreator],
		description: "For automation: it consists of the permissions to create Sensors in secured clusters",
		resourceWithAccess: []permissions.ResourceWithAccess{
			permissions.View(resources.Cluster),
			permissions.Modify(resources.Cluster),
			permissions.Modify(resources.Administration),
		},
	},
	accesscontrol.VulnMgmtApprover: {
		idSuffix:    "vulnmgmtapprover",
		postgresID:  accesscontrol.DefaultPermissionSetIDs[accesscontrol.VulnMgmtApprover],
		description: "For users: use it to provide access to approve vulnerability deferrals or false positive requests",
		resourceWithAccess: []permissions.ResourceWithAccess{
			permissions.View(resources.VulnerabilityManagementApprovals),
			permissions.Modify(resources.VulnerabilityManagementApprovals),
		},
	},
	accesscontrol.VulnMgmtRequester: {
		idSuffix:    "vulnmgmtrequester",
		postgresID:  accesscontrol.DefaultPermissionSetIDs[accesscontrol.VulnMgmtRequester],
		description: "For users: use it to provide access to request vulnerability deferrals or false positives",
		resourceWithAccess: []permissions.ResourceWithAccess{
			permissions.View(resources.VulnerabilityManagementRequests),
			permissions.Modify(resources.VulnerabilityManagementRequests),
		},
	},
	accesscontrol.VulnReporter: {
		idSuffix:    "vulnreporter",
		postgresID:  accesscontrol.DefaultPermissionSetIDs[accesscontrol.VulnReporter],
		description: "For users: use it to create and manage vulnerability reporting configurations for scheduled vulnerability reports",
		resourceWithAccess: func() []permissions.ResourceWithAccess {
			return []permissions.ResourceWithAccess{
				permissions.View(resources.WorkflowAdministration),   // required for vuln report configurations
				permissions.Modify(resources.WorkflowAdministration), // required for vuln report configurations
				permissions.View(resources.Integration),              // required for vuln report configurations
			}
		}(),
	},
	accesscontrol.VulnerabilityManagementConsumer: {
		idSuffix:    "vulnmgmtconsumer",
		postgresID:  accesscontrol.DefaultPermissionSetIDs[accesscontrol.VulnerabilityManagementConsumer],
		description: "For users: use it to provide read-only access to analyze vulnerabilities and initiate risk acceptance process",
		resourceWithAccess: []permissions.ResourceWithAccess{
			permissions.View(resources.Node),
			permissions.View(resources.Deployment),
			permissions.View(resources.Image),
			permissions.View(resources.WatchedImage),
			permissions.View(resources.WorkflowAdministration),
			permissions.Modify(resources.VulnerabilityManagementRequests),
		},
	},
	accesscontrol.VulnerabilityManagementAdmin: {
		idSuffix:    "vulnmgmtadmin",
		postgresID:  accesscontrol.DefaultPermissionSetIDs[accesscontrol.VulnerabilityManagementAdmin],
		description: "For users: Use it to provide administrative access to analyze vulnerabilities, generate reports, and manage risk acceptance process",
		resourceWithAccess: []permissions.ResourceWithAccess{
			permissions.View(resources.Cluster),
			permissions.View(resources.Node),
			permissions.View(resources.Namespace),
			permissions.View(resources.Deployment),
			permissions.View(resources.Image),
			permissions.View(resources.Integration),
			permissions.Modify(resources.WatchedImage),
			permissions.Modify(resources.VulnerabilityManagementRequests),
			permissions.Modify(resources.VulnerabilityManagementApprovals),
			permissions.Modify(resources.WorkflowAdministration),
		},
	},
}

func getDefaultRoles() []*storage.Role {
	roles := make([]*storage.Role, 0, len(defaultPermissionSets))
	for _, roleName := range accesscontrol.DefaultRoleNames.AsSlice() {
		attributes, found := defaultPermissionSets[roleName]
		if !found {
			utils.Should(errors.Errorf("Default role %s does not have permission set defined", roleName))
		}

		// Historically, default permission sets and roles have had same name to make it easier for the user to quickly map one to the other.
		permissionSet := getDefaultPermissionSet(roleName)
		if permissionSet == nil {
			utils.Should(errors.Errorf("No default permission set found for default role %s", roleName))
			continue
		}

		role := &storage.Role{
			Name:          roleName,
			Description:   attributes.description,
			AccessScopeId: rolePkg.AccessScopeIncludeAll.GetId(),
			Traits: &storage.Traits{
				Origin: storage.Traits_DEFAULT,
			},
			PermissionSetId: permissionSet.Id,
		}
		roles = append(roles, role)
	}
	return roles
}

func getDefaultPermissionSets() []*storage.PermissionSet {
	permissionSets := make([]*storage.PermissionSet, 0, len(defaultPermissionSets))

	for name, attributes := range defaultPermissionSets {
		resourceToAccess := permissionsUtils.FromResourcesWithAccess(attributes.resourceWithAccess...)

		permissionSet := &storage.PermissionSet{
			Id:               attributes.getID(),
			Name:             name,
			Description:      attributes.description,
			ResourceToAccess: resourceToAccess,
			Traits: &storage.Traits{
				Origin: storage.Traits_DEFAULT,
			},
		}
		permissionSets = append(permissionSets, permissionSet)
	}

	return permissionSets
}

func getDefaultPermissionSet(name string) *storage.PermissionSet {
	_, found := accesscontrol.DefaultPermissionSetIDs[name]
	if !found {
		return nil
	}

	attributes, found := defaultPermissionSets[name]
	if !found {
		return nil
	}

	return &storage.PermissionSet{
		Id:               attributes.getID(),
		Name:             name,
		Description:      attributes.description,
		ResourceToAccess: permissionsUtils.FromResourcesWithAccess(attributes.resourceWithAccess...),
		Traits: &storage.Traits{
			Origin: storage.Traits_DEFAULT,
		},
	}
}

func getDefaultAccessScopes() []*storage.SimpleAccessScope {
	return []*storage.SimpleAccessScope{
		rolePkg.AccessScopeIncludeAll,
		rolePkg.AccessScopeExcludeAll,
	}
}
