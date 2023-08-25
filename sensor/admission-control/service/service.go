package service

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	pkgGRPC "github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/grpc/authz/allow"
	"github.com/stackrox/rox/pkg/grpc/routes"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/sensor/admission-control/manager"
	"google.golang.org/grpc"
	admission "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sRuntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/json"
)

var (
	log = logging.LoggerForModule()

	universalDeserializer = serializer.NewCodecFactory(k8sRuntime.NewScheme()).UniversalDeserializer()
)

type service struct {
	mgr manager.Manager
}

// New creates a new admission control API service
func New(mgr manager.Manager) pkgGRPC.APIService {
	return &service{
		mgr: mgr,
	}
}

func (s *service) RegisterServiceHandler(context.Context, *runtime.ServeMux, *grpc.ClientConn) error {
	return nil
}

func (s *service) RegisterServiceServer(*grpc.Server) {}

func (s *service) CustomRoutes() []routes.CustomRoute {
	return []routes.CustomRoute{
		{
			Route:         "/ready",
			Authorizer:    allow.Anonymous(),
			ServerHandler: http.HandlerFunc(s.handleReady),
			Compression:   false,
		},
		{
			Route:         "/validate",
			Authorizer:    allow.Anonymous(),
			ServerHandler: http.HandlerFunc(s.handleValidate),
			Compression:   false,
		},
		{
			Route:         "/events",
			Authorizer:    allow.Anonymous(),
			ServerHandler: http.HandlerFunc(s.handleK8sEvents),
			Compression:   false,
		},
	}
}

func (s *service) handleReady(w http.ResponseWriter, _ *http.Request) {
	if !s.mgr.IsReady() {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = fmt.Fprintln(w, "not ready")
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprintln(w, "ok")
}

func (s *service) handleRequest(w http.ResponseWriter, req *http.Request, validateFunc func(*admission.AdmissionRequest) (*admission.AdmissionResponse, error)) {
	if req.Method == http.MethodGet {
		_, _ = fmt.Fprintln(w, "{}")
		return
	}

	if req.Method != http.MethodPost {
		http.Error(w, "Endpoint only supports GET and POST requests", http.StatusBadRequest)
		return
	}

	if !s.mgr.IsReady() {
		http.Error(w, "No settings are available. Not ready to handle admission controller requests", http.StatusServiceUnavailable)
		return
	}

	admissionRequest, apiVersion, err := readAdmissionRequest(req)
	if err != nil {
		log.Errorf("Failed to read admission request body: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	reviewResp, err := validateFunc(admissionRequest)
	if err != nil {
		log.Errorf("Error handling admission request: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := &admission.AdmissionReview{
		Response: reviewResp,
		TypeMeta: metav1.TypeMeta{
			APIVersion: apiVersion,
			Kind:       "AdmissionReview",
		},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		log.Errorf("Could not marshal admission review response to JSON: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Debugf("Sending admission review: %s", data)

	if _, err := w.Write(data); err != nil {
		log.Errorf("Could not send admission review response back to client: %v", err)
	}
}

func (s *service) handleValidate(w http.ResponseWriter, req *http.Request) {
	s.handleRequest(w, req, s.mgr.HandleValidate)
}

func (s *service) handleK8sEvents(w http.ResponseWriter, req *http.Request) {
	s.handleRequest(w, req, s.mgr.HandleK8sEvent)
}

func readAdmissionRequest(req *http.Request) (*admission.AdmissionRequest, string, error) {
	respBody, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, "", errors.Wrap(err, "reading request body")
	}

	log.Debugf("Received admission review request: %s", respBody)

	var admissionReview admission.AdmissionReview
	if _, _, err := universalDeserializer.Decode(respBody, nil, &admissionReview); err != nil {
		return nil, "", errors.Wrap(err, "decoding v1 admission review")
	}

	return admissionReview.Request, admissionReview.APIVersion, nil
}
