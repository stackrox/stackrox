package profiler

import (
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"net/http"
	"os"
	"time"

	"connectrpc.com/connect"
	googlev1 "github.com/grafana/pyroscope/api/gen/proto/go/google/v1"
	v1 "github.com/grafana/pyroscope/api/gen/proto/go/querier/v1"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/utils"
)

const (
	selectMergeProfile = "/querier.v1.QuerierService/SelectMergeProfile"
)

var (
	log = logging.LoggerForModule()

	querierServiceServiceDescriptor                  = v1.File_querier_v1_querier_proto.Services().ByName("QuerierService")
	querierServiceSelectMergeProfileMethodDescriptor = querierServiceServiceDescriptor.Methods().ByName("SelectMergeProfile")
)

type Client interface {
	Query(string) (*QueryResponse, error)
}

func NewClient() Client {
	return newPyroscopeClient()
}

type pyroscopeClient struct {
	url string
}

func newPyroscopeClient() *pyroscopeClient {
	return &pyroscopeClient{
		url: "http://localhost:4040",
	}
}

func (p *pyroscopeClient) Query(query string) (*QueryResponse, error) {
	httpClient := &http.Client{}
	client := connect.NewClient[v1.SelectMergeProfileRequest, googlev1.Profile](
		httpClient,
		p.url+selectMergeProfile,
		connect.WithSchema(querierServiceSelectMergeProfileMethodDescriptor),
	)
	req := &v1.SelectMergeProfileRequest{
		ProfileTypeID: "memory:inuse_space:bytes:space:bytes",
		Start:         time.Now().UnixMilli() - time.Hour.Milliseconds(),
		End:           time.Now().UnixMilli(),
		LabelSelector: query,
	}
	res, err := client.CallUnary(context.Background(), connect.NewRequest(req))
	if err != nil {
		log.Errorf("could not query pyroscope: %v", err)
		return nil, err
	}
	buf, err := res.Msg.MarshalVT()
	if err != nil {
		log.Errorf("unable to marshal the profile: %v", err)
		return nil, err
	}
	f, err := os.OpenFile("temp-heap.pprof.gz", os.O_RDWR|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		log.Errorf("unable to create file: %v", err)
		return nil, err
	}
	defer utils.IgnoreError(f.Close)
	gzWriter := gzip.NewWriter(f)
	defer utils.IgnoreError(gzWriter.Close)
	if _, err := io.Copy(gzWriter, bytes.NewReader(buf)); err != nil {
		log.Errorf("unable to write profile: %v", err)
		return nil, err
	}
	log.Info("===================================== DONE")
	return &QueryResponse{}, nil
}
