package create

import (
	"bytes"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/cloudflare/cfssl/helpers"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/authproviders/userpki"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/roxctl/common"
)

const (
	short = "Create a new user certificate authority provider"
	long  = short + "\n" + `
Uses the supplied PEM-encoded root certificate files to create a new
user certificate authentication provider.`
)

var (
	flagPEMFiles      []string
	flagRoleName      string
	errNoPEMFiles     = errors.New("no certificate files specified")
	errNotCA          = errors.New("not a certificate authority")
	errNoProviderName = errors.New("no provider name specified")
)

// Command adds the userpki create command
func Command() *cobra.Command {
	c := &cobra.Command{
		Use:   "create name",
		Short: short,
		Long:  long,
		RunE:  createProvider,
	}
	c.Flags().StringSliceVarP(&flagPEMFiles, "cert", "c", nil, "Root CA certificate PEM files (can supply multiple)")
	utils.Must(c.MarkFlagRequired("cert"))
	c.Flags().StringVarP(&flagRoleName, "role", "r", "", "Default role for provider")
	utils.Must(c.MarkFlagRequired("role"))
	return c
}

func isSelfSigned(cert *x509.Certificate) bool {
	return bytes.Equal(cert.RawSubject, cert.RawIssuer)
}

func createProvider(c *cobra.Command, args []string) error {
	if len(flagPEMFiles) == 0 {
		return errNoPEMFiles
	}
	if len(args) != 1 {
		return errNoProviderName
	}
	providerName := args[0]

	var pems strings.Builder
	for _, fn := range flagPEMFiles {
		b, err := ioutil.ReadFile(fn)
		if err != nil {
			return errors.Wrap(err, fn)
		}
		cert, err := helpers.ParseCertificatePEM(b)
		if err != nil {
			return errors.Wrap(err, fn)
		}
		if !cert.IsCA && !isSelfSigned(cert) {
			return errors.Wrap(errNotCA, fn)
		}
		_, err = pems.Write(b)
		utils.Must(err)
		utils.Must(pems.WriteByte('\n'))
	}

	conn, err := common.GetGRPCConnection()
	if err != nil {
		return err
	}
	defer utils.IgnoreError(conn.Close)
	ctx := common.Context()

	authService := v1.NewAuthProviderServiceClient(conn)
	groupService := v1.NewGroupServiceClient(conn)
	roleService := v1.NewRoleServiceClient(conn)
	_, err = roleService.GetRole(ctx, &v1.ResourceByID{Id: flagRoleName})
	if err != nil {
		return errors.Wrap(err, flagRoleName)
	}

	req := &v1.PostAuthProviderRequest{
		Provider: &storage.AuthProvider{
			Type:    userpki.TypeName,
			Name:    providerName,
			Enabled: true,
			Config: map[string]string{
				userpki.ConfigKeys: pems.String(),
			},
		},
	}
	provider, err := authService.PostAuthProvider(ctx, req)
	if err != nil {
		return err
	}

	_, err = groupService.CreateGroup(ctx, &storage.Group{
		Props: &storage.GroupProperties{
			AuthProviderId: provider.GetId(),
		},
		RoleName: flagRoleName,
	})

	if err != nil {
		return err
	}

	fmt.Printf("Provider created with ID %s\n", provider.GetId())
	return nil
}
