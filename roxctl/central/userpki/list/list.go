package list

import (
	"fmt"
	"os"

	"github.com/cloudflare/cfssl/helpers"
	"github.com/golang/protobuf/jsonpb"
	"github.com/spf13/cobra"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/authproviders/clientca"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/roxctl/common"
)

var (
	json bool
)

// Command adds the userpki list command
func Command() *cobra.Command {
	c := &cobra.Command{
		Use:   "list",
		Short: "List all user certificate authority providers",
		Long:  "List all user certificate authority providers",
		RunE:  listProviders,
	}
	c.Flags().BoolVarP(&json, "json", "j", false, "JSON output")
	return c
}

func listProviders(cmd *cobra.Command, args []string) error {
	conn, err := common.GetGRPCConnection()
	if err != nil {
		return err
	}
	defer utils.IgnoreError(conn.Close)
	ctx := common.Context()

	authClient := v1.NewAuthProviderServiceClient(conn)
	groupClient := v1.NewGroupServiceClient(conn)
	providers, err := authClient.GetAuthProviders(ctx, &v1.GetAuthProvidersRequest{Type: clientca.TypeName})
	if err != nil {
		return err
	}
	if json {
		m := jsonpb.Marshaler{Indent: "  "}
		err = m.Marshal(os.Stdout, providers)
		if err == nil {
			fmt.Println()
		}
		return err
	}
	if len(providers.GetAuthProviders()) == 0 {
		fmt.Println("No user certificate providers configured")
		return nil
	}
	groups, err := groupClient.GetGroups(ctx, &v1.Empty{})
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
		PrintProviderDetails(p, defaultRoles)
	}
	return nil
}

// PrintProviderDetails print the details of a provider.
func PrintProviderDetails(p *storage.AuthProvider, defaultRoles map[string]string) {
	fmt.Printf("Provider: %s\n", p.GetName())
	fmt.Printf("  ID: %s\n", p.GetId())
	fmt.Printf("  Enabled: %t\n", p.GetEnabled())
	if len(defaultRoles) > 0 {
		fmt.Printf("  Default role: %q\n", defaultRoles[p.GetId()])
	}
	pem := p.GetConfig()[clientca.ConfigKeys]
	certs, err := helpers.ParseCertificatesPEM([]byte(pem))
	if err != nil {
		fmt.Printf("  Certificates: %v\n", err)
		return
	}
	if len(certs) == 0 {
		fmt.Printf("  Certificates: none\n")
	}
	for i, cert := range certs {
		fmt.Printf("  Certificate %d:\n", i+1)
		fmt.Printf("    DN: %s\n", cert.Subject.String())
		fmt.Printf("    Expiration: %s\n", cert.NotAfter)
	}
}
