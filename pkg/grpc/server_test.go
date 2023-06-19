package grpc

import (
	"os"
	"testing"

	"github.com/stackrox/rox/pkg/mtls/verifier"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMaxResponseMsgSize_Unset(t *testing.T) {
	require.NoError(t, os.Unsetenv(maxResponseMsgSizeSetting.EnvVar()))

	assert.Equal(t, defaultMaxResponseMsgSize, maxResponseMsgSize())
}

func TestMaxResponseMsgSize_Empty(t *testing.T) {
	require.NoError(t, os.Setenv(maxResponseMsgSizeSetting.EnvVar(), ""))

	assert.Equal(t, defaultMaxResponseMsgSize, maxResponseMsgSize())
}

func TestMaxResponseMsgSize_Invalid(t *testing.T) {
	require.NoError(t, os.Setenv(maxResponseMsgSizeSetting.EnvVar(), "notAnInt"))

	assert.Equal(t, defaultMaxResponseMsgSize, maxResponseMsgSize())
}

func TestMaxResponseMsgSize_Valid(t *testing.T) {
	require.NoError(t, os.Setenv(maxResponseMsgSizeSetting.EnvVar(), "1337"))

	assert.Equal(t, 1337, maxResponseMsgSize())
}

func Test_NewAPI(t *testing.T) {
	// TODO: Use TLS mock instead of overriding this with dummy certs
	utils.CrashOnError(os.Setenv("ROX_MTLS_CERT_FILE", "../../tools/local-sensor/certs/cert.pem"))
	utils.CrashOnError(os.Setenv("ROX_MTLS_KEY_FILE", "../../tools/local-sensor/certs/key.pem"))
	utils.CrashOnError(os.Setenv("ROX_MTLS_CA_FILE", "../../tools/local-sensor/certs/caCert.pem"))
	utils.CrashOnError(os.Setenv("ROX_MTLS_CA_KEY_FILE", "../../tools/local-sensor/certs/caKey.pem"))

	conf := Config{
		Endpoints: []*EndpointConfig{
			{
				ListenEndpoint: ":8080",
				TLS:            verifier.NonCA{},
				ServeGRPC:      true,
				ServeHTTP:      true,
			},
		},
	}

	api := NewAPI(conf)

	started := api.Start()
	started.Wait()
}

func Test_NewAPI_StartAndStop(t *testing.T) {
	// TODO: Use TLS mock instead of overriding this with dummy certs
	utils.CrashOnError(os.Setenv("ROX_MTLS_CERT_FILE", "../../tools/local-sensor/certs/cert.pem"))
	utils.CrashOnError(os.Setenv("ROX_MTLS_KEY_FILE", "../../tools/local-sensor/certs/key.pem"))
	utils.CrashOnError(os.Setenv("ROX_MTLS_CA_FILE", "../../tools/local-sensor/certs/caCert.pem"))
	utils.CrashOnError(os.Setenv("ROX_MTLS_CA_KEY_FILE", "../../tools/local-sensor/certs/caKey.pem"))

	conf := Config{
		Endpoints: []*EndpointConfig{
			{
				ListenEndpoint: ":8080",
				TLS:            verifier.NonCA{},
				ServeGRPC:      true,
				ServeHTTP:      true,
			},
		},
	}

	api := NewAPI(conf)

	t.Logf("Starting API")
	started := api.Start()
	started.Wait()

	t.Logf("Stopping API")
	stopped := api.Stop()
	t.Logf("Waiting to fully stop API")
	stopped.Wait()

	t.Logf("Create new API with same conf")

	newApi := NewAPI(conf)
	restarted := newApi.Start()
	restarted.Wait()

}
