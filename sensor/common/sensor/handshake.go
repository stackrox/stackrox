package sensor

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ProbeStreamForConnectionError probes a stream via Recv() to retrieve the
// actual server-side error when central did not echo the SensorHello metadata
// key. revokedMsg is a phrase describing the revoked credential (e.g.
// "the init bundle credentials have been revoked" or "the cluster registration
// secret has been revoked") and noEchoMsg is the error message returned when
// Recv() succeeds without an error, meaning central is reachable but did not
// acknowledge the handshake.
func ProbeStreamForConnectionError(stream central.SensorService_CommunicateClient, revokedMsg, noEchoMsg string) error {
	if _, recvErr := stream.Recv(); recvErr != nil {
		if st, ok := status.FromError(recvErr); ok && st.Code() == codes.Unauthenticated {
			return errors.Wrapf(recvErr, "central rejected the connection, possibly because"+
				" %s in central."+
				" Check central logs for details", revokedMsg)
		}
		return errors.Wrap(recvErr, "central rejected the connection")
	}
	return errors.New(noEchoMsg)
}
