package upload

import (
	"bytes"
	"context"
	"os"
	"strings"
	"time"

	"github.com/golang/protobuf/jsonpb"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/policies"
	"github.com/stackrox/rox/pkg/registries/types"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/flags"
)

// Command provides the upload command for policies to OCI registries.
func Command(cliEnvironment environment.Environment) *cobra.Command {
	upload := uploadCmd{env: cliEnvironment}

	cmd := &cobra.Command{
		Use:   "upload",
		Short: "Upload ACS policies as OCI artifacts to a registry",
		Long: `Upload ACS policies as OCI artifacts to a registry.
Policies that have been exported can be uploaded as OCI artifacts to a registry of your choice, allowing you to easily
share them between environments or with communities.

You need to have the following to use the command:
- R/W access to the repository where ACS policies are uploaded.
- A registry that supports OCI artifacts, in particular also custom artifacts (looking at you Quay).
- An exported policy.

Afterwards, you simply need to pass the reference of the repository where policies should be uploaded, the file reference,
and your credentials.

After upload, you will receive the digest of the policy. You may verify the uploaded OCI artifact by using e.g. crane:
crane manifest <your-reference>:<returned-digest-from-roxctl> | jq

As a sample, you should see something along the lines of:
{
  "schemaVersion": 2,
  "mediaType": "application/vnd.oci.image.manifest.v1+json",
  "artifactType": "application/vnd.stackrox.policy",
  "config": {
    "mediaType": "application/vnd.oci.empty.v1+json",
    "digest": "sha256:44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a",
    "size": 2,
    "data": "e30="
  },
  "layers": [
    {
      "mediaType": "text/json",
      "digest": "sha256:4a5dfca072b52ba8dd47d5a0623efe73044ce533b7f60455b2f9dca8a7247ef4",
      "size": 831,
      "annotations": {
        "org.opencontainers.image.title": "c19c0cea-b5df-40c4-80e7-836a1b0785e6"
      }
    }
  ],
  "annotations": {
    "org.opencontainers.image.created": "2023-09-26T06:01:48Z"
  }
}
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := upload.Construct(cmd); err != nil {
				return err
			}
			return upload.Upload()
		},
	}

	cmd.Flags().StringVar(&upload.username, "username", "", "user name to use for registry access")
	cmd.Flags().StringVar(&upload.password, "password", "", "password to use for registry access")
	cmd.Flags().StringVar(&upload.reference, "reference", "",
		"reference to the repository where the policy should be uploaded. MUST be in format <registry host>/<repository>..")
	cmd.Flags().StringVarP(&upload.file, "file", "f", "",
		"file containing the exported policy in JSON format")

	utils.Should(cmd.MarkFlagRequired("username"))
	utils.Should(cmd.MarkFlagRequired("password"))
	utils.Should(cmd.MarkFlagRequired("reference"))
	utils.Should(cmd.MarkFlagRequired("file"))

	flags.AddTimeout(cmd)

	flags.HideInheritedFlags(cmd)

	return cmd
}

type uploadCmd struct {
	env environment.Environment

	pusher policies.Pusher

	username     string
	password     string
	reference    string
	file         string
	fileContents []byte
	timeout      time.Duration
}

func (u *uploadCmd) Construct(cmd *cobra.Command) error {
	u.timeout = flags.Timeout(cmd)

	userName, err := cmd.Flags().GetString("username")
	if err != nil {
		return errors.Wrap(err, "couldn't get username flag")
	}
	u.username = userName

	password, err := cmd.Flags().GetString("password")
	if err != nil {
		return errors.Wrap(err, "couldn't get password flag")
	}
	u.password = password

	ref, err := cmd.Flags().GetString("reference")
	if err != nil {
		return errors.Wrap(err, "couldn't get reference flag")
	}
	u.reference = ref

	file, err := cmd.Flags().GetString("file")
	if err != nil {
		return errors.Wrap(err, "couldn't get file flag")
	}
	u.file = file

	contents, err := os.ReadFile(u.file)
	if err != nil {
		return errors.Wrapf(err, "reading file %q", u.file)
	}
	u.fileContents = contents

	u.pusher = policies.NewPusher()

	return nil
}

func (u *uploadCmd) Upload() error {
	// Create the registry config from flag values.
	referenceSplit := strings.SplitN(u.reference, "/", 2)
	registryHostname := referenceSplit[0]
	repository := referenceSplit[1]
	registryConfig := &types.Config{
		Username:         u.username,
		Password:         u.password,
		RegistryHostname: registryHostname,
	}
	var policy storage.Policy
	if err := jsonpb.Unmarshal(bytes.NewReader(u.fileContents), &policy); err != nil {
		return errors.Wrap(err, "unmarshalling policy")
	}

	ctx, cancel := context.WithTimeout(context.Background(), u.timeout)
	defer cancel()

	digest, err := u.pusher.Push(ctx, &policy, registryConfig, repository)
	if err != nil {
		return errors.Wrap(err, "pushing policy to registry")
	}

	u.env.Logger().PrintfLn("Successfully uploaded policy to reference %s:%s", u.reference, digest)
	return nil
}
