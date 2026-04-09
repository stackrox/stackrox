package sensor

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const sensorHelloNotAcknowledgedMsg = "central did not acknowledge SensorHello," +
	" likely due to a networking or TLS configuration issue" +
	" (e.g., re-encrypt routes or TLS termination)" +
	" preventing central from receiving sensor's TLS certificate"

// ProbeStreamForConnectionError probes a stream via Recv() to retrieve the
// actual server-side error when central did not echo the SensorHello metadata
// key. deniedMsg is the suggestion shown when central returns PermissionDenied
// (e.g. revoked or expired credentials). For any other error, or when Recv()
// succeeds without an error, a generic networking/TLS suggestion is returned.
func ProbeStreamForConnectionError(stream central.SensorService_CommunicateClient, deniedMsg string) error {
	if _, recvErr := stream.Recv(); recvErr != nil {
		if st, ok := status.FromError(recvErr); ok && st.Code() == codes.PermissionDenied {
			return errors.Wrapf(recvErr, "central rejected the connection: %s."+
				" Check central logs for details", deniedMsg)
		}
		return errors.Wrap(recvErr, sensorHelloNotAcknowledgedMsg)
	}
	return errors.New(sensorHelloNotAcknowledgedMsg)
}
