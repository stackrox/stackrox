package liveprobe

import (
	"context"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

type ProbedCert struct {
	Subject         string    `json:"subject"`
	Issuer          string    `json:"issuer"`
	SANs            []string  `json:"sans,omitempty"`
	NotBefore       time.Time `json:"notBefore"`
	NotAfter        time.Time `json:"notAfter"`
	IsExpired       bool      `json:"isExpired"`
	IsCA            bool      `json:"isCA"`
	Algorithm       string    `json:"algorithm"`
	PostQuantumSafe bool      `json:"postQuantumSafe"`
	Fingerprint     string    `json:"fingerprint"`
}

type ProbeResult struct {
	ServiceName   string      `json:"serviceName"`
	Namespace     string      `json:"namespace"`
	Port          int         `json:"port"`
	Endpoint      string      `json:"endpoint"`
	Cert          *ProbedCert `json:"cert,omitempty"`
	SecretMatch   string      `json:"secretMatch,omitempty"`
	MatchedSecret string      `json:"matchedSecret,omitempty"`
	Error         string      `json:"error,omitempty"`
}

func probeViaPortForward(ctx context.Context, restConfig *rest.Config, client kubernetes.Interface, namespace string, svc ServiceDef) *ProbeResult {
	result := &ProbeResult{
		ServiceName: svc.Name,
		Namespace:   namespace,
		Port:        svc.Port,
		Endpoint:    "port-forward",
	}

	pod, containerPort, err := findReadyPod(ctx, client, namespace, svc.Name, svc.Port)
	if err != nil {
		result.Error = fmt.Sprintf("finding pod: %v", err)
		return result
	}

	localPort, stop, err := startPortForward(restConfig, pod, containerPort)
	if err != nil {
		result.Error = fmt.Sprintf("port-forward: %v", err)
		return result
	}
	defer close(stop)

	addr := fmt.Sprintf("127.0.0.1:%d", localPort)
	cert, err := probeTLS(ctx, addr, svc.Protocol)
	if err != nil {
		result.Error = fmt.Sprintf("TLS probe: %v", err)
		return result
	}

	result.Cert = buildProbedCert(cert)
	return result
}

func probeDirectTLS(ctx context.Context, addr string, svc ServiceDef) *ProbeResult {
	result := &ProbeResult{
		ServiceName: svc.Name,
		Namespace:   "",
		Port:        svc.Port,
		Endpoint:    addr,
	}

	cert, err := probeTLS(ctx, addr, svc.Protocol)
	if err != nil {
		result.Error = fmt.Sprintf("TLS probe: %v", err)
		return result
	}

	result.Cert = buildProbedCert(cert)
	return result
}

func findReadyPod(ctx context.Context, client kubernetes.Interface, namespace, serviceName string, servicePort int) (*corev1.Pod, int32, error) {
	svc, err := client.CoreV1().Services(namespace).Get(ctx, serviceName, metav1.GetOptions{})
	if err != nil {
		return nil, 0, fmt.Errorf("getting service %s/%s: %w", namespace, serviceName, err)
	}

	var targetPortNum int32
	var targetPortName string
	for _, p := range svc.Spec.Ports {
		if int(p.Port) == servicePort {
			if p.TargetPort.IntValue() != 0 {
				targetPortNum = int32(p.TargetPort.IntValue())
			} else if p.TargetPort.String() != "" && p.TargetPort.String() != "0" {
				targetPortName = p.TargetPort.String()
			} else {
				targetPortNum = p.Port
			}
			break
		}
	}
	if targetPortNum == 0 && targetPortName == "" {
		return nil, 0, fmt.Errorf("port %d not found on service %s/%s", servicePort, namespace, serviceName)
	}

	selector := labels.Set(svc.Spec.Selector).String()
	pods, err := client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: selector,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("listing pods for %s/%s: %w", namespace, serviceName, err)
	}

	for i := range pods.Items {
		pod := &pods.Items[i]
		if pod.Status.Phase != corev1.PodRunning {
			continue
		}
		for _, c := range pod.Status.Conditions {
			if c.Type == corev1.PodReady && c.Status == corev1.ConditionTrue {
				if targetPortName != "" {
					resolved, err := resolveNamedPort(pod, targetPortName)
					if err != nil {
						return nil, 0, err
					}
					return pod, resolved, nil
				}
				return pod, targetPortNum, nil
			}
		}
	}

	return nil, 0, fmt.Errorf("no ready pod found for service %s/%s", namespace, serviceName)
}

func resolveNamedPort(pod *corev1.Pod, portName string) (int32, error) {
	for _, c := range pod.Spec.Containers {
		for _, p := range c.Ports {
			if p.Name == portName {
				return p.ContainerPort, nil
			}
		}
	}
	return 0, fmt.Errorf("named port %q not found in pod %s/%s", portName, pod.Namespace, pod.Name)
}

func startPortForward(restConfig *rest.Config, pod *corev1.Pod, targetPort int32) (uint16, chan struct{}, error) {
	transport, upgrader, err := spdy.RoundTripperFor(restConfig)
	if err != nil {
		return 0, nil, fmt.Errorf("configuring transport: %w", err)
	}

	reqURL, err := url.Parse(restConfig.Host)
	if err != nil {
		return 0, nil, fmt.Errorf("parsing API server URL: %w", err)
	}
	reqURL.Path = fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/portforward", pod.Namespace, pod.Name)

	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, http.MethodPost, reqURL)

	stopCh := make(chan struct{})
	readyCh := make(chan struct{})

	ports := []string{fmt.Sprintf("0:%d", targetPort)}
	fw, err := portforward.New(dialer, ports, stopCh, readyCh, nil, nil)
	if err != nil {
		return 0, nil, fmt.Errorf("creating port-forwarder: %w", err)
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- fw.ForwardPorts()
	}()

	select {
	case <-readyCh:
	case err := <-errCh:
		return 0, nil, fmt.Errorf("port-forward failed: %w", err)
	}

	fwPorts, err := fw.GetPorts()
	if err != nil {
		close(stopCh)
		return 0, nil, fmt.Errorf("getting forwarded ports: %w", err)
	}

	return fwPorts[0].Local, stopCh, nil
}

func probeTLS(ctx context.Context, addr, protocol string) (*x509.Certificate, error) {
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("connecting to %s: %w", addr, err)
	}
	defer conn.Close()

	if protocol == "postgres" {
		if err := postgresSSLRequest(conn); err != nil {
			return nil, err
		}
	}

	tlsConn := tls.Client(conn, &tls.Config{
		InsecureSkipVerify: true,
	})
	if err := tlsConn.HandshakeContext(ctx); err != nil {
		return nil, fmt.Errorf("TLS handshake: %w", err)
	}
	defer tlsConn.Close()

	certs := tlsConn.ConnectionState().PeerCertificates
	if len(certs) == 0 {
		return nil, fmt.Errorf("server returned no certificates")
	}

	return certs[0], nil
}

// postgresSSLRequest performs the PostgreSQL SSLRequest handshake.
// Pattern from central/credentialexpiry/service/service_impl.go.
func postgresSSLRequest(conn net.Conn) error {
	if err := binary.Write(conn, binary.BigEndian, []int32{8, 80877103}); err != nil {
		return fmt.Errorf("sending SSLRequest: %w", err)
	}
	response := make([]byte, 1)
	if _, err := io.ReadFull(conn, response); err != nil {
		return fmt.Errorf("reading SSLRequest response: %w", err)
	}
	if response[0] != 'S' {
		return fmt.Errorf("server refused TLS (response: %c)", response[0])
	}
	return nil
}

func buildProbedCert(cert *x509.Certificate) *ProbedCert {
	fp := sha256.Sum256(cert.Raw)

	var sans []string
	for _, dns := range cert.DNSNames {
		sans = append(sans, dns)
	}
	for _, ip := range cert.IPAddresses {
		sans = append(sans, ip.String())
	}

	return &ProbedCert{
		Subject:         cert.Subject.String(),
		Issuer:          cert.Issuer.String(),
		SANs:            sans,
		NotBefore:       cert.NotBefore,
		NotAfter:        cert.NotAfter,
		IsExpired:       time.Now().After(cert.NotAfter),
		IsCA:            cert.IsCA,
		Algorithm:       describeAlgorithm(cert),
		PostQuantumSafe: false,
		Fingerprint:     hex.EncodeToString(fp[:]),
	}
}

func describeAlgorithm(cert *x509.Certificate) string {
	switch pub := cert.PublicKey.(type) {
	case *ecdsa.PublicKey:
		return fmt.Sprintf("ECDSA %s", pub.Curve.Params().Name)
	case *rsa.PublicKey:
		return fmt.Sprintf("RSA %d", pub.N.BitLen())
	case ed25519.PublicKey:
		return "Ed25519"
	default:
		return cert.PublicKeyAlgorithm.String()
	}
}
