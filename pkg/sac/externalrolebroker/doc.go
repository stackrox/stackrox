// Package externalrolebroker provides utilities for managing external role information
// from sources like ACM (Advanced Cluster Management) UserPermissions.
//
// Example usage with ACM client:
//
//	import (
//		"context"
//
//		"github.com/stackrox/rox/pkg/sac/externalrolebroker"
//		"github.com/stackrox/rox/pkg/sac/externalrolebroker/acmclient"
//	)
//
//	func example() error {
//		// Create ACM client
//		client, err := acmclient.NewACMClient()
//		if err != nil {
//			return err
//		}
//
//		// Method 1: Get ResolvedRoles directly (recommended)
//		// This function handles listing, filtering, and conversion in one call
//		resolvedRoles, err := externalrolebroker.GetResolvedRolesFromACM(context.Background(), client)
//		if err != nil {
//			return err
//		}
//
//		// Each ResolvedRole contains:
//		// - RoleName: from UserPermission metadata name
//		// - Permissions: map of ACS resources to access levels
//		// - AccessScope: clusters and namespaces where permissions apply
//		for _, role := range resolvedRoles {
//			_ = role.GetRoleName()      // e.g., "cluster-admin"
//			_ = role.GetPermissions()   // e.g., {Namespace: READ_WRITE_ACCESS}
//			_ = role.GetAccessScope()   // e.g., {IncludedClusters: ["cluster-a"]}
//		}
//
//		// Method 2: Manual step-by-step conversion (for advanced use cases)
//		// You can also manually invoke each step if you need more control
//		/*
//			allPermissions, err := client.ListUserPermissions(context.Background(), metav1.ListOptions{})
//			if err != nil {
//				return err
//			}
//
//			baseK8sPermissions := externalrolebroker.FilterUserPermissionsForBaseK8sResources(allPermissions.Items)
//
//			for _, permission := range baseK8sPermissions {
//				permissionSet := externalrolebroker.ConvertClusterRoleToPermissionSet(
//					permission.Status.ClusterRoleDefinition,
//				)
//				accessScope := externalrolebroker.ConvertBindingsToSimpleAccessScope(
//					permission.Status.Bindings,
//				)
//				// Create ResolvedRole manually using roletest.NewResolvedRole
//			}
//		*/
//
//		return nil
//	}
package externalrolebroker
