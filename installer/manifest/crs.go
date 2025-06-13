package manifest

import (
	"bufio"
	"context"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	stackroxv1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/retry"
	"github.com/stackrox/rox/pkg/size"
	"github.com/stackrox/rox/pkg/utils"

	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

type CRSGenerator struct{}

func (g CRSGenerator) Name() string {
	return "Cluster Registration Service (CRS)"
}

func (g CRSGenerator) Exportable() bool {
	return false
}

func (g CRSGenerator) Generate(ctx context.Context, m *manifestGenerator) ([]Resource, error) {
	if m.Config.Action == "apply" {
		sec, err := m.Client.CoreV1().Secrets(m.Config.Namespace).Get(ctx, "cluster-registration-secret", metav1.GetOptions{})
		if sec != nil && err == nil {
			log.Info("CRS already exists")
			return []Resource{}, nil
		} else if !k8serrors.IsNotFound(err) {
			return []Resource{}, errors.Wrap(err, "Failed to fetch cluster-registration-secret")
		}
	}

	port, err := getRandomPort()
	if err != nil {
		panic(err)
	}

	var resp *stackroxv1.CRSGenResponse
	err = retry.WithRetry(func() error {
		pfCloseChan, err := g.portForward(ctx, port, m)
		if err != nil {
			return errors.Wrap(err, "Failed to create port forwarder")
		}
		defer close(pfCloseChan)

		log.Info("Creating grpc connection to central...")

		conn, err := g.centralConnection(ctx, port, m)
		if err != nil {
			return errors.Wrap(err, "Failed to create gRPC connection")
		}
		defer utils.IgnoreError(conn.Close)

		log.Info("Invoking CRS endpoint...")
		svc := stackroxv1.NewClusterInitServiceClient(conn)
		req := stackroxv1.CRSGenRequest{Name: "local"}
		resp, err = svc.GenerateCRS(ctx, &req)
		if err != nil {
			errStatus, ok := status.FromError(err)
			if !ok && !strings.Contains(err.Error(), "certificate errors:") {
				return retry.MakeRetryable(err)
			}
			if ok && errStatus.Code() == codes.Unimplemented {
				return errors.Wrap(err, "missing CRS support in Central")
			}
			return errors.Wrap(err, "generating new CRS")
		}
		return nil
	}, retry.Tries(30), retry.BetweenAttempts(func(previousAttemptNumber int) {
		log.Info("Waiting for central endpoint to start listening")
		time.Sleep(5 * time.Second)
	}), retry.OnlyRetryableErrors())

	if err != nil {
		return []Resource{}, errors.Wrap(err, "Failed to retrieve CRS from Central")
	}

	crs := extractCRS(resp.GetCrs())

	if crs == "" {
		return []Resource{}, errors.New("Failed to extract CRS")
	}

	data, err := base64.StdEncoding.DecodeString(crs)
	if err != nil {
		return []Resource{}, errors.Wrap(err, "Failed to base64 decode CRS")
	}

	crsSecret := v1.Secret{
		Data: map[string][]byte{"crs": []byte(data)},
	}
	crsSecret.SetName("cluster-registration-secret")
	crsSecret.SetGroupVersionKind(v1.SchemeGroupVersion.WithKind("Secret"))

	return []Resource{{
		Object:       &crsSecret,
		Name:         crsSecret.Name,
		IsUpdateable: false,
	}}, nil
}

func getRandomPort() (int, error) {
	listener, err := net.Listen("tcp", ":0") // 0 means a random port will be assigned
	if err != nil {
		return 0, err
	}
	defer listener.Close()
	_, port, _ := net.SplitHostPort(listener.Addr().String())
	return strconv.Atoi(port)
}

func (g *CRSGenerator) portForward(ctx context.Context, port int, m *manifestGenerator) (chan struct{}, error) {
	pods, err := m.Client.CoreV1().Pods(m.Config.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: metav1.FormatLabelSelector(&metav1.LabelSelector{MatchLabels: map[string]string{"app": "central"}}),
	})
	if err != nil {
		return nil, err
	} else if len(pods.Items) == 0 {
		return nil, fmt.Errorf("no central pods found: %v", err)
	}

	centralPod := pods.Items[0]

	if centralPod.Status.Phase != v1.PodRunning {
		return nil, errors.New("Pod not yet ready")
	}

	podName := centralPod.Name

	if podName == "" {
		return nil, errors.New("Timed out waiting for pod to start running")
	}

	req := m.Client.CoreV1().RESTClient().Post().
		Resource("pods").
		Namespace(m.Config.Namespace).
		Name(podName).
		SubResource("portforward")

	transport, upgrader, err := spdy.RoundTripperFor(m.RESTConfig)
	if err != nil {
		return nil, err
	}

	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, http.MethodPost, req.URL())
	if err != nil {
		return nil, fmt.Errorf("Failed to create websocket: %v", err)
	}

	stopChan := make(chan struct{}, 1)
	readyChan := make(chan struct{})

	pf, err := portforward.New(dialer, []string{fmt.Sprintf("%d:8443", port)}, stopChan, readyChan, os.Stdout, os.Stderr)
	if err != nil {
		return nil, fmt.Errorf("failed to create tunnel for port-forwarding: %v", err)
	}

	go func() {
		err = pf.ForwardPorts()
	}()

	log.Info("Waiting for port forwarder to become ready...")

	select {
	case <-readyChan:
		log.Info("port forwarder ready!")
		return stopChan, nil
	case <-time.After(5 * time.Second):
		errMsg := "Timed out waiting for port forwarding to start"
		if err != nil {
			return nil, errors.Wrap(err, errMsg)
		}
		return nil, errors.New(errMsg)
	}
}

func (g *CRSGenerator) centralConnection(ctx context.Context, port int, m *manifestGenerator) (*grpc.ClientConn, error) {
	clientconn.SetUserAgent("Installer")

	dialOpts := []grpc.DialOption{
		grpc.WithNoProxy(),
	}

	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(m.CA.CertPEM())

	opts := clientconn.Options{
		InsecureNoTLS:                  false,
		InsecureAllowCredsViaPlaintext: false,
		DialOptions:                    dialOpts,
		TLS: clientconn.TLSConfigOptions{
			RootCAs: pool,
		},
	}

	opts.ConfigureBasicAuth("admin", "letmein")

	callOpts := []grpc.CallOption{grpc.MaxCallRecvMsgSize(12 * size.MB)}
	centralHostPort := fmt.Sprintf("localhost:%d", port)
	return clientconn.GRPCConnection(ctx, mtls.CentralSubject, centralHostPort, opts, grpc.WithDefaultCallOptions(callOpts...))
}

func extractCRS(input []byte) string {
	scanner := bufio.NewScanner(strings.NewReader(string(input)))

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "  crs: ") {
			return strings.TrimPrefix(line, "  crs: ")
		}
	}
	return ""
}

func init() {
	crs = append(crs, CRSGenerator{})
}
