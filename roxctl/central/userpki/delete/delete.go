package delete

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/authproviders/userpki"
	"github.com/stackrox/rox/pkg/errox"
	pkgCommon "github.com/stackrox/rox/pkg/roxctl/common"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stackrox/rox/roxctl/central/userpki/list"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/flags"
)

var (
	errNoProviderArg    = errox.InvalidArgs.New("provider ID/name parameter required")
	errProviderNotFound = errox.NotFound.New("provider doesn't exist")
)

type centralUserPkiDeleteCommand struct {
	// Properties that are bound to cobra flags.
	providerArg string

	// Properties that are injected or constructed.
	env          environment.Environment
	timeout      time.Duration
	retryTimeout time.Duration
}

// Command adds the userpki delete command.
func Command(cliEnvironment environment.Environment) *cobra.Command {
	c := &cobra.Command{
		Use:   "delete id|name",
		Short: "Delete a user certificate authentication provider.",
		Long:  "Delete a configured user certificate authentication provider and its associated group mappings.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return errNoProviderArg
			}
			centralUserPkiDeleteCommand := makeCentralUserPkiDeleteCommand(cliEnvironment, cmd, args)
			deleteProvider, err := centralUserPkiDeleteCommand.prepareDeleteProvider()
			if err != nil {
				return err
			}
			if err := flags.CheckConfirmation(cmd, cliEnvironment.Logger(), cliEnvironment.InputOutput()); err != nil {
				return err
			}
			return deleteProvider()
		},
	}
	flags.AddForce(c)
	flags.AddTimeout(c)
	flags.AddRetryTimeout(c)
	return c
}

func makeCentralUserPkiDeleteCommand(cliEnvironment environment.Environment, cmd *cobra.Command, args []string) *centralUserPkiDeleteCommand {
	return &centralUserPkiDeleteCommand{
		providerArg:  args[0],
		env:          cliEnvironment,
		timeout:      flags.Timeout(cmd),
		retryTimeout: flags.RetryTimeout(cmd),
	}
}

func getAuthProviderByID(ctx context.Context, svc v1.AuthProviderServiceClient, id string) (*storage.AuthProvider, error) {
	return svc.GetAuthProvider(ctx, &v1.GetAuthProviderRequest{Id: id})
}

func getAuthProviderByName(ctx context.Context, svc v1.AuthProviderServiceClient, name string) (*storage.AuthProvider, error) {
	provs, err := svc.GetAuthProviders(ctx, &v1.GetAuthProvidersRequest{Name: name, Type: userpki.TypeName})
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

func (cmd *centralUserPkiDeleteCommand) prepareDeleteProvider() (func() error, error) {
	conn, err := cmd.env.GRPCConnection(cmd.retryTimeout)
	if err != nil {
		return nil, err
	}
	defer utils.IgnoreError(conn.Close)
	ctx, cancel := context.WithTimeout(pkgCommon.Context(), cmd.timeout)
	defer cancel()
	authService := v1.NewAuthProviderServiceClient(conn)
	groupService := v1.NewGroupServiceClient(conn)

	var prov *storage.AuthProvider

	_, err = uuid.FromString(cmd.providerArg)
	if err == nil {
		prov, err = getAuthProviderByID(ctx, authService, cmd.providerArg)
	} else {
		prov, err = getAuthProviderByName(ctx, authService, cmd.providerArg)
	}

	if err != nil {
		return nil, err
	}
	group, err := groupService.GetGroup(ctx, &storage.GroupProperties{AuthProviderId: prov.GetId()})

	defaultRoles := make(map[string]string)
	if err == nil && group != nil {
		defaultRoles[prov.GetId()] = group.GetRoleName()
	}
	list.PrintProviderDetails(cmd.env.Logger(), prov, defaultRoles)

	return func() error {
		cmd.env.Logger().PrintfLn("Deleting provider and rolemappings.")

		_, err := authService.DeleteAuthProvider(ctx, &v1.DeleteByIDWithForce{
			Id: prov.GetId(),
		})
		if err != nil {
			return err
		}

		groups, err := groupService.GetGroups(ctx, &v1.GetGroupsRequest{})
		if err != nil {
			return err
		}
		var relevantGroups []*storage.Group
		for _, v := range groups.GetGroups() {
			if v.GetProps().GetAuthProviderId() == cmd.providerArg {
				relevantGroups = append(relevantGroups, v)
			}
		}
		if len(relevantGroups) != 0 {
			_, err := groupService.BatchUpdate(ctx, &v1.GroupBatchUpdateRequest{
				PreviousGroups: relevantGroups,
				RequiredGroups: nil,
			})
			if err != nil {
				return err
			}
		}
		cmd.env.Logger().PrintfLn("Successfully deleted.")
		return nil
	}, nil
}
