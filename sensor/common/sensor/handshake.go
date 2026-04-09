package sensor

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ProbeStreamForConnectionError probes a stream via Recv() to retrieve the
// actual server-side error when central did not echo the SensorHello metadata
// key. deniedMsg is the suggestion shown when central returns PermissionDenied
// (e.g. revoked or expired credentials). genericMsg is the fallback error
// message used for any other error or when Recv() succeeds without an error.
func ProbeStreamForConnectionError(stream central.SensorService_CommunicateClient, deniedMsg, genericMsg string) error {
	if _, recvErr := stream.Recv(); recvErr != nil {
		if st, ok := status.FromError(recvErr); ok && st.Code() == codes.PermissionDenied {
			return errors.Wrapf(recvErr, "central rejected the connection: %s."+
				" Check central logs for details", deniedMsg)
		}
		return errors.Wrap(recvErr, genericMsg)
	}
	return errors.New(genericMsg)
}
