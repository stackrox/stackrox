package sensor

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ProbeStreamForConnectionError probes a stream via Recv() to retrieve the
// actual server-side error when central did not echo the SensorHello metadata
// key. credentialDesc should describe the type of credential used for the
// connection (e.g. "init bundle credentials" or "the cluster registration
// secret") and noEchoMsg is the error message returned when Recv() succeeds
// without an error, meaning central is reachable but did not acknowledge the
// handshake.
func ProbeStreamForConnectionError(stream central.SensorService_CommunicateClient, credentialDesc, noEchoMsg string) error {
	if _, recvErr := stream.Recv(); recvErr != nil {
		if st, ok := status.FromError(recvErr); ok && st.Code() == codes.Unauthenticated {
			return errors.Wrapf(recvErr, "central rejected the connection, possibly because"+
				" %s has been revoked in central."+
				" Check central logs for details", credentialDesc)
		}
		return errors.Wrap(recvErr, "central rejected the connection")
	}
	return errors.New(noEchoMsg)
}
