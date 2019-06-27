package delete

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/authproviders/clientca"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stackrox/rox/roxctl/central/userpki/list"
	"github.com/stackrox/rox/roxctl/common"
	"github.com/stackrox/rox/roxctl/common/flags"
)

const (
	short = "Delete a user certificate authentication provider"
	long  = short + "\n\n" + "This will remove the configured authentication provider and all associated group mappings"
)

var (
	errNoProviderArg    = errors.New("provider ID/name parameter required")
	errProviderNotFound = errors.New("provider doesn't exist")
)

// Command adds the userpki delete command
func Command() *cobra.Command {

	c := &cobra.Command{
		Use:   "delete id|name",
		Short: short,
		Long:  long,
		RunE:  deleteProvider,
	}
	flags.AddForce(c)
	return c
}

func getAuthProviderByID(ctx context.Context, svc v1.AuthProviderServiceClient, id string) (*storage.AuthProvider, error) {
	return svc.GetAuthProvider(ctx, &v1.GetAuthProviderRequest{Id: id})
}

func getAuthProviderByName(ctx context.Context, svc v1.AuthProviderServiceClient, name string) (*storage.AuthProvider, error) {
	provs, err := svc.GetAuthProviders(ctx, &v1.GetAuthProvidersRequest{Name: name, Type: clientca.TypeName})
	if err != nil {
		return nil, err
	}
	all := provs.GetAuthProviders()
	if len(all) == 0 {
		return nil, errProviderNotFound
	}
	if len(all) > 1 {
		return nil, fmt.Errorf("%d providers by that name, use id", len(provs.GetAuthProviders()))
	}
	return all[0], nil
}

func deleteProvider(c *cobra.Command, args []string) error {
	if len(args) != 1 {
		return errNoProviderArg
	}
	providerArg := args[0]

	conn, err := common.GetGRPCConnection()
	if err != nil {
		return err
	}
	defer utils.IgnoreError(conn.Close)
	ctx := common.Context()
	authService := v1.NewAuthProviderServiceClient(conn)
	groupService := v1.NewGroupServiceClient(conn)

	var prov *storage.AuthProvider

	_, err = uuid.FromString(providerArg)
	if err == nil {
		prov, err = getAuthProviderByID(ctx, authService, providerArg)
	} else {
		prov, err = getAuthProviderByName(ctx, authService, providerArg)
	}

	if err != nil {
		return err
	}
	group, err := groupService.GetGroup(ctx, &storage.GroupProperties{AuthProviderId: prov.GetId()})

	defaultRoles := make(map[string]string)
	if err == nil && group != nil {
		defaultRoles[prov.GetId()] = group.GetRoleName()
	}
	list.PrintProviderDetails(prov, defaultRoles)
	fmt.Println("Deleting provider and rolemappings.")

	err = flags.CheckConfirmation(c)
	if err != nil {
		return err
	}

	_, err = authService.DeleteAuthProvider(ctx, &v1.ResourceByID{
		Id: prov.GetId(),
	})

	if err != nil {
		return err
	}

	groups, err := groupService.GetGroups(ctx, &v1.Empty{})
	if err != nil {
		return err
	}
	var relevantGroups []*storage.Group
	for _, v := range groups.GetGroups() {
		if v.GetProps().GetAuthProviderId() == providerArg {
			relevantGroups = append(relevantGroups, v)
		}
	}
	if len(relevantGroups) == 0 {
		fmt.Println("Successfully deleted.")
		return nil
	}
	_, err = groupService.BatchUpdate(ctx, &v1.GroupBatchUpdateRequest{
		PreviousGroups: relevantGroups,
		RequiredGroups: nil,
	})

	if err != nil {
		return err
	}
	fmt.Println("Successfully deleted.")

	return err
}
