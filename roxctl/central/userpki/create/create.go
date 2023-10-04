package create

import (
	"bytes"
	"context"
	"crypto/x509"
	"os"
	"strings"
	"time"

	"github.com/cloudflare/cfssl/helpers"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/authproviders/userpki"
	"github.com/stackrox/rox/pkg/errox"
	pkgCommon "github.com/stackrox/rox/pkg/roxctl/common"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/flags"
)

type centralUserPkiCreateCommand struct {
	// Properties that are bound to cobra flags.
	pemFiles []string
	roleName string

	// Properties that are injected or constructed.
	env          environment.Environment
	timeout      time.Duration
	retryTimeout time.Duration
	providerName string
}

var (
	errNoPEMFiles     = errox.InvalidArgs.New("no certificate files specified")
	errNotCA          = errox.InvalidArgs.New("not a certificate authority")
	errNoProviderName = errox.InvalidArgs.New("no provider name specified")
)

// Command adds the userpki create command
func Command(cliEnvironment environment.Environment) *cobra.Command {
	centralUserPkiCreateCmd := &centralUserPkiCreateCommand{env: cliEnvironment}
	c := &cobra.Command{
		Use:   "create name",
		Short: "Create a new user certificate authentication provider.",
		Long:  "Create a new user certificate authentication provider by using the provided PEM-encoded root certificate files.",
		RunE: func(c *cobra.Command, args []string) error {
			if err := centralUserPkiCreateCmd.validate(args); err != nil {
				return err
			}
			if err := centralUserPkiCreateCmd.construct(c, args); err != nil {
				return err
			}
			return centralUserPkiCreateCmd.createProvider()
		},
	}
	c.Flags().StringSliceVarP(&centralUserPkiCreateCmd.pemFiles, "cert", "c", nil, "Root CA certificate PEM files (can supply multiple)")
	utils.Must(c.MarkFlagRequired("cert"))
	c.Flags().StringVarP(&centralUserPkiCreateCmd.roleName, "role", "r", "", "Minimum access role for users of this provider")
	utils.Must(c.MarkFlagRequired("role"))
	flags.AddTimeout(c)
	flags.AddRetryTimeout(c)
	return c
}

func (cmd *centralUserPkiCreateCommand) validate(args []string) error {
	if len(cmd.pemFiles) == 0 {
		return errNoPEMFiles
	}
	if len(args) != 1 {
		return errNoProviderName
	}
	return nil
}

func (cmd *centralUserPkiCreateCommand) construct(cbr *cobra.Command, args []string) error {
	cmd.providerName = args[0]
	cmd.timeout = flags.Timeout(cbr)
	cmd.retryTimeout = flags.RetryTimeout(cbr)
	return nil
}

func isSelfSigned(cert *x509.Certificate) bool {
	return bytes.Equal(cert.RawSubject, cert.RawIssuer)
}

func (cmd *centralUserPkiCreateCommand) createProvider() error {
	var pems strings.Builder
	for _, fn := range cmd.pemFiles {
		b, err := os.ReadFile(fn)
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
		utils.CrashOnError(err)
		utils.Must(pems.WriteByte('\n'))
	}

	conn, err := cmd.env.GRPCConnection(cmd.retryTimeout)
	if err != nil {
		return err
	}
	defer utils.IgnoreError(conn.Close)
	ctx, cancel := context.WithTimeout(pkgCommon.Context(), cmd.timeout)
	defer cancel()

	authService := v1.NewAuthProviderServiceClient(conn)
	groupService := v1.NewGroupServiceClient(conn)
	roleService := v1.NewRoleServiceClient(conn)
	_, err = roleService.GetRole(ctx, &v1.ResourceByID{Id: cmd.roleName})
	if err != nil {
		return errors.Wrap(err, cmd.roleName)
	}

	req := &v1.PostAuthProviderRequest{
		Provider: &storage.AuthProvider{
			Type:    userpki.TypeName,
			Name:    cmd.providerName,
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
		RoleName: cmd.roleName,
	})

	if err != nil {
		return err
	}

	cmd.env.Logger().PrintfLn("Provider created with ID %s", provider.GetId())
	return nil
}
