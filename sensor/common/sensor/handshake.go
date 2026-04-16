package sensor

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const networkingOrTLSMsg = "Connection to Central failed," +
	" likely due to a networking or TLS configuration issue" +
	" (e.g., re-encrypt routes or TLS termination)."

// DiagnoseConnectionFailure probes the stream to retrieve the actual server-side error
// when central did not acknowledge the SensorHello message. It logs a user-actionable
// message (credential issue for PermissionDenied, networking/TLS suggestion otherwise)
// and returns an error.
func DiagnoseConnectionFailure(stream central.SensorService_CommunicateClient, deniedMsg string) error {
	_, err := stream.Recv()

	if err != nil {
		if st, ok := status.FromError(err); ok && st.Code() == codes.PermissionDenied {
			log.Errorf("Central rejected the connection: %s. Check central logs for details.", deniedMsg)
			return errors.Wrap(err, "permission denied by central")
		}
	} else {
		err = errors.New("central did not acknowledge SensorHello")
	}

	log.Error(networkingOrTLSMsg)
	return err
}
