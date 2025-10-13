package main

import (
	"context"
	"fmt"
	"os"
	"path"

	"github.com/stackrox/rox/central/metadata/service"
	"github.com/stackrox/rox/central/tlsconfig"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/jsonutil"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/utils"
)

const usageMsg = "Usage: go run main.go <challenge_token> <central_certs_path> <additional_ca_path>"

// Usage: go run exec_tlschallenge.go <challenge_token> <central_certs_path> <additional_ca_path>
func main() {
	// Check if a command-line argument is provided
	if len(os.Args) < 4 {
		fmt.Println(usageMsg) //nolint:forbidigo
		os.Exit(1)
	}

	challengeToken := os.Args[1]
	certPath := os.Args[2]
	additionalCAPath := os.Args[3]

	utils.Should(os.Setenv(mtls.CAFileEnvName, path.Join(certPath, "ca.pem")))
	utils.Should(os.Setenv(tlsconfig.MTLSAdditionalCADirEnvName, additionalCAPath))
	utils.Should(os.Setenv(mtls.CertFilePathEnvName, path.Join(certPath, "cert.pem")))
	utils.Should(os.Setenv(mtls.KeyFileEnvName, path.Join(certPath, "key.pem")))

	result, err := tlsChallenge(challengeToken, certPath, additionalCAPath)
	if err != nil {
		fmt.Printf("Error unmarshalling Protobuf data: %v\n", err) //nolint:forbidigo
		os.Exit(1)
	}
	fmt.Printf("%+v\n", result) //nolint:forbidigo
}

func tlsChallenge(challengeToken, certPath, additionalCAPath string) (string, error) {
	utils.Should(os.Setenv(mtls.CAFileEnvName, path.Join(certPath, "ca.pem")))
	utils.Should(os.Setenv(tlsconfig.MTLSAdditionalCADirEnvName, additionalCAPath))
	utils.Should(os.Setenv(mtls.CertFilePathEnvName, path.Join(certPath, "cert.pem")))
	utils.Should(os.Setenv(mtls.KeyFileEnvName, path.Join(certPath, "key.pem")))

	metadataService := service.New()
	message, err := metadataService.TLSChallenge(context.Background(), &v1.TLSChallengeRequest{
		ChallengeToken: challengeToken,
	})
	if err != nil {
		return "", fmt.Errorf("Failed marshalling %T TLSChallenge: %w\n", message, err)
	}

	result, err := jsonutil.ProtoToJSON(message)
	if err != nil {
		return "", fmt.Errorf("error unmarshalling Protobuf data: %w", err)
	}
	return result, nil
}
