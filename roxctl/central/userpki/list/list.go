package list

import (
	"context"
	"time"

	"github.com/cloudflare/cfssl/helpers"
	"github.com/golang/protobuf/jsonpb"
	"github.com/spf13/cobra"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/authproviders/userpki"
	pkgCommon "github.com/stackrox/rox/pkg/roxctl/common"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/roxctl/common"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/flags"
	"github.com/stackrox/rox/roxctl/common/logger"
)

type centralUserPkiListCommand struct {
	// Properties that are bound to cobra flags.
	json bool

	// Properties that are injected or constructed.
	env          environment.Environment
	timeout      time.Duration
	retryTimeout time.Duration
}

// Command adds the userpki list command
func Command(cliEnvironment environment.Environment) *cobra.Command {
	centralUserPkiListCmd := &centralUserPkiListCommand{env: cliEnvironment}
	c := &cobra.Command{
		Use:   "list",
		Short: "Display all user certificate authentication providers.",
		Long:  "Display all configured user certificate authentication providers in a human-readable or JSON format.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := centralUserPkiListCmd.construct(cmd); err != nil {
				return err
			}
			return centralUserPkiListCmd.listProviders()
		},
	}
	c.Flags().BoolVarP(&centralUserPkiListCmd.json, "json", "j", false, "Enable JSON output")
	flags.AddTimeout(c)
	flags.AddRetryTimeout(c)
	return c
}

func (cmd *centralUserPkiListCommand) construct(cbr *cobra.Command) error {
	cmd.timeout = flags.Timeout(cbr)
	cmd.retryTimeout = flags.RetryTimeout(cbr)
	return nil
}

func (cmd *centralUserPkiListCommand) listProviders() error {
	conn, err := cmd.env.GRPCConnection(common.WithRetryTimeout(cmd.retryTimeout))
	if err != nil {
		return err
	}
	defer utils.IgnoreError(conn.Close)

	ctx, cancel := context.WithTimeout(pkgCommon.Context(), cmd.timeout)
	defer cancel()

	authClient := v1.NewAuthProviderServiceClient(conn)
	groupClient := v1.NewGroupServiceClient(conn)
	providers, err := authClient.GetAuthProviders(ctx, &v1.GetAuthProvidersRequest{Type: userpki.TypeName})
	if err != nil {
		return err
	}
	if cmd.json {
		m := jsonpb.Marshaler{Indent: "  "}
		err = m.Marshal(cmd.env.InputOutput().Out(), providers)
		if err == nil {
			cmd.env.Logger().PrintfLn("")
		}
		return err
	}
	if len(providers.GetAuthProviders()) == 0 {
		cmd.env.Logger().InfofLn("No user certificate providers configured")
		return nil
	}
	groups, err := groupClient.GetGroups(ctx, &v1.GetGroupsRequest{})
	if err != nil {
		return err
	}
	defaultRoles := make(map[string]string)
	for _, g := range groups.GetGroups() {
		id := g.GetProps().GetAuthProviderId()
		if id != "" && g.GetProps().GetKey() == "" {
			defaultRoles[id] = g.GetRoleName()
		}
	}

	for _, p := range providers.GetAuthProviders() {
		PrintProviderDetails(cmd.env.Logger(), p, defaultRoles)
	}
	return nil
}

// PrintProviderDetails print the details of a provider.
func PrintProviderDetails(logger logger.Logger, p *storage.AuthProvider, defaultRoles map[string]string) {
	logger.PrintfLn("Provider: %s", p.GetName())
	logger.PrintfLn("  ID: %s", p.GetId())
	logger.PrintfLn("  Enabled: %t", p.GetEnabled())
	if len(defaultRoles) > 0 {
		logger.PrintfLn("  Minimum access role: %q", defaultRoles[p.GetId()])
	}
	pem := p.GetConfig()[userpki.ConfigKeys]
	certs, err := helpers.ParseCertificatesPEM([]byte(pem))
	if err != nil {
		logger.PrintfLn("  Certificates: %v", err)
		return
	}
	if len(certs) == 0 {
		logger.PrintfLn("  Certificates: none")
	}
	for i, cert := range certs {
		logger.PrintfLn("  Certificate %d:", i+1)
		logger.PrintfLn("    DN: %s", cert.Subject.String())
		logger.PrintfLn("    Expiration: %s", cert.NotAfter)
	}
}
