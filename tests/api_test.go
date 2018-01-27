package tests

import (
	"context"
	"testing"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/clientconn"
	"github.com/golang/protobuf/ptypes/empty"
)

func TestPing(t *testing.T) {
	conn, err := clientconn.UnauthenticatedGRPCConnection(localEndpoint)
	if err != nil {
		t.Fatal(err)
	}

	s := v1.NewPingServiceClient(conn)
	_, err = s.Ping(context.Background(), &empty.Empty{})
	if err != nil {
		t.Error(err)
	}
}
