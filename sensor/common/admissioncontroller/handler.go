package admissioncontroller

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"text/template"
	"time"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/enforcers"
	"github.com/stackrox/rox/pkg/kubernetes"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/protoconv/resources"
	"github.com/stackrox/rox/pkg/stringutils"
	"github.com/stackrox/rox/pkg/templates"
	"google.golang.org/grpc"
	admission "k8s.io/api/admission/v1beta1"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	timeout = 30 * time.Second

	// This purposefully leaves a newline at the top for formatting when using kubectl
	kubectlTemplate = `
Policy: {{.Title}}
In case of emergency, add the annotation {"admission.stackrox.io/break-glass": "ticket-1234"} to your deployment with an updated ticket number
{{range .Alerts}}
{{.Policy.Name}}
- Description:
    ↳ {{wrap .Policy.Description}}
- Rationale:
    ↳ {{wrap .Policy.Rationale}}
- Remediation:
    ↳ {{wrap .Policy.Remediation}}
- Violations:
    {{- range .Violations}}
    - {{.Message}}
    {{- end}}
{{ end}}
`
)

var (
	log = logging.LoggerForModule()

	msgTemplate = template.Must(template.New("name").Funcs(
		template.FuncMap{
			"wrap": stringutils.Wrap,
		}).Parse(kubectlTemplate))
)

// NewHandler returns a handler that proxies admission controllers to Central
func NewHandler(conn *grpc.ClientConn) http.Handler {
	return &handlerImpl{
		client: v1.NewDetectionServiceClient(conn),
	}
}

type handlerImpl struct {
	client v1.DetectionServiceClient
}

func admissionPass(w http.ResponseWriter, id types.UID) {
	writeResponse(w, id, true, "")
}

func writeResponse(w http.ResponseWriter, id types.UID, allowed bool, reason string) {
	var ar *admission.AdmissionReview
	if allowed {
		ar = &admission.AdmissionReview{
			Response: &admission.AdmissionResponse{
				UID:     id,
				Allowed: true,
			},
		}
	} else {
		ar = &admission.AdmissionReview{
			Response: &admission.AdmissionResponse{
				UID:     id,
				Allowed: false,
				Result: &metav1.Status{
					Status:  "Failure",
					Reason:  metav1.StatusReason("Failed currently enforced policies from StackRox"),
					Message: reason,
				},
			},
		}
	}

	data, err := json.Marshal(ar)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if _, err := w.Write(data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func parseIntoDeployment(ar *admission.AdmissionReview) (*storage.Deployment, error) {
	var objType interface{}
	if ar.Request == nil {
		return nil, nil
	}
	switch ar.Request.Kind.Kind {
	case kubernetes.Pod:
		objType = &core.Pod{}
	case kubernetes.Deployment:
		objType = &apps.Deployment{}
	case kubernetes.StatefulSet:
		objType = &apps.StatefulSet{}
	case kubernetes.DaemonSet:
		objType = &apps.DaemonSet{}
	case kubernetes.ReplicationController:
		objType = &core.ReplicationController{}
	case kubernetes.ReplicaSet:
		objType = &apps.ReplicaSet{}
	default:
		log.Errorf("Currently do not recognize kind %q in admission controller", ar.Request.Kind.Kind)
		return nil, nil
	}

	if err := json.Unmarshal(ar.Request.Object.Raw, &objType); err != nil {
		return nil, err
	}

	return resources.NewDeploymentFromStaticResource(objType, ar.Request.Kind.Kind)
}

// ServeHTTP serves the admission controller endpoint
func (s *handlerImpl) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Endpoint only supports POST requests", http.StatusBadRequest)
		return
	}

	decoder := json.NewDecoder(r.Body)
	var admissionReview admission.AdmissionReview
	if err := decoder.Decode(&admissionReview); err != nil {
		log.Errorf("Error decoding admission review: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	deployment, err := parseIntoDeployment(&admissionReview)
	if err != nil {
		log.Errorf("Error parsing into deployment: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if deployment == nil {
		admissionPass(w, admissionReview.Request.UID)
		return
	}

	if !enforcers.ShouldEnforce(deployment.GetAnnotations()) {
		log.Warnf("Deployment %s/%s of type %s was deployed without being checked due to emergency annotations",
			deployment.GetNamespace(), deployment.GetName(), deployment.GetType())
		admissionPass(w, admissionReview.Request.UID)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	resp, err := s.client.DetectDeployTime(ctx, &v1.DeployDetectionRequest{Resource: &v1.DeployDetectionRequest_Deployment{Deployment: deployment}})
	if err != nil {
		log.Warnf("Deployment %s/%s of type %s was deployed without being checked due to detection error: %v",
			deployment.GetNamespace(), deployment.GetName(), deployment.GetType(), err)
		admissionPass(w, admissionReview.Request.UID)
		return
	}

	var enforcedAlerts []*storage.Alert
	var totalPolicies int
	// There will only ever be one run in this call
	for _, r := range resp.GetRuns() {
		for _, a := range r.GetAlerts() {
			totalPolicies++
			for _, e := range a.GetPolicy().GetEnforcementActions() {
				if e == storage.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT ||
					e == storage.EnforcementAction_UNSATISFIABLE_NODE_CONSTRAINT_ENFORCEMENT {
					enforcedAlerts = append(enforcedAlerts, a)
					break
				}
			}
		}
	}

	var topMsg string
	if len(enforcedAlerts) == 0 {
		admissionPass(w, admissionReview.Request.UID)
		return
	}
	if len(enforcedAlerts) == 1 {
		topMsg = fmt.Sprintf("Violated %d policies total. 1 enforced policy is described below:", totalPolicies)
	} else {
		topMsg = fmt.Sprintf("Violated %d policies total. %d enforced policies are described below:", totalPolicies, len(enforcedAlerts))
	}

	msg, err := templates.ExecuteToString(msgTemplate, map[string]interface{}{
		"Title":  topMsg,
		"Alerts": enforcedAlerts,
	})
	if err != nil {
		log.Errorf("Error executing msg template: %v", err)
		admissionPass(w, admissionReview.Request.UID)
		return
	}
	writeResponse(w, admissionReview.Request.UID, false, msg)
}
