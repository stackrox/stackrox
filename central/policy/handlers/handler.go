package handlers

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/pkg/errors"
	notifierDatastore "github.com/stackrox/rox/central/notifier/datastore"
	"github.com/stackrox/rox/central/policy/customresource"
	policyDatastore "github.com/stackrox/rox/central/policy/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/apiparams"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/jsonutil"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/protoadapt"
)

var (
	log                                      = logging.LoggerForModule()
	maxCustomResourceNameToFilePathTruncSize = customresource.MaxCustomResourceMetadataNameLength - uuid.StringLength - 1
)

// Handler returns a handler for policy http requests
func Handler(p policyDatastore.DataStore, n notifierDatastore.DataStore) http.Handler {
	return httpHandler{
		policyStore:   p,
		notifierStore: n,
	}
}

type httpHandler struct {
	policyStore   policyDatastore.DataStore
	notifierStore notifierDatastore.DataStore
}

func (h httpHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writer.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var params apiparams.SaveAsCustomResourcesRequest
	err := json.NewDecoder(request.Body).Decode(&params)
	if err != nil {
		httputil.WriteGRPCStyleError(writer, codes.Internal, err)
		return
	}
	h.saveAsCustomResources(request.Context(), &params, writer)
}

// saveAsCustomResources saves the policies, designed by the policy ids, as custom resources
func (h httpHandler) saveAsCustomResources(ctx context.Context, request *apiparams.SaveAsCustomResourcesRequest, writer http.ResponseWriter) {
	policyList, missingIndices, err := h.policyStore.GetPolicies(ctx, request.IDs)
	if err != nil {
		httputil.WriteGRPCStyleError(writer, codes.Internal, err)
		return
	}
	errDetails := &v1.PolicyOperationErrorList{}
	for _, missingIndex := range missingIndices {
		policyID := request.IDs[missingIndex]
		errDetails.Errors = append(errDetails.Errors, &v1.PolicyOperationError{
			PolicyId: policyID,
			Error: &v1.PolicyError{
				Error: "not found",
			},
		})
		log.Errorf("A policy error occurred for id %s: not found", policyID)
	}
	if len(errDetails.GetErrors()) > 0 {
		writeErrorWithDetails(writer, codes.InvalidArgument, errors.New("Failed to retrieve all policies. Check error details for a list of policies that could not be retrieved."), errDetails)
		return
	}

	notifiers := make(map[string]string)
	err = h.notifierStore.ProcessNotifiers(ctx, func(n *storage.Notifier) error {
		notifiers[n.GetId()] = n.GetName()
		return nil
	})
	if err != nil {
		httputil.WriteGRPCStyleError(writer, codes.Internal, err)
		return
	}

	zipWriter := zip.NewWriter(writer)
	defer utils.IgnoreError(zipWriter.Close)

	writer.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="security_policies_%s.zip"`, time.Now().Format(time.RFC3339)))
	writer.Header().Set("Content-Type", "application/zip")
	names := set.NewStringSet()
	for _, policy := range policyList {
		cr := customresource.ConvertPolicyToCustomResource(policy)
		// Switch notifier IDs to names
		notifierNames := make([]string, 0, len(cr.SecurityPolicySpec.Notifiers))
		for _, id := range cr.SecurityPolicySpec.Notifiers {
			if name, exists := notifiers[id]; exists {
				notifierNames = append(notifierNames, name)
				continue
			}
			log.Errorf("Notifier %s in policy %s not found, hence skipped", id, policy.GetName())
		}
		cr.SecurityPolicySpec.Notifiers = notifierNames

		// Rename custom resource if its name conflicts with existing resource in the zip archive.
		crName, ok := cr.Metadata["name"].(string)
		if !ok {
			err := utils.ShouldErr(errors.Errorf("Unexpected custom resource without metadata name: %+v", cr))
			httputil.WriteGRPCStyleError(writer, codes.Internal, err)
			break
		}
		if !names.Add(crName) {
			crName = subDNSDomainToZipFileName(crName, policy.GetId())
			cr.Metadata["name"] = crName
		}

		fileName := fmt.Sprintf("%s.yaml", crName)
		// Write to zip archive
		crWriter, err := zipWriter.Create(fileName)
		if err != nil {
			httputil.WriteGRPCStyleError(writer, codes.Unavailable, errors.Wrapf(err, "error creating %s in zip", fileName))
			break
		}
		err = customresource.WriteCustomResource(crWriter, cr)
		if err != nil {
			errDetails.Errors = append(errDetails.Errors, &v1.PolicyOperationError{
				PolicyId: policy.GetId(),
				Error: &v1.PolicyError{
					Error: errors.Wrap(err, "Failed to marshal policy to custom resource").Error(),
				},
			})
		}
	}
	if len(errDetails.GetErrors()) > 0 {
		writeErrorWithDetails(writer, codes.InvalidArgument, errors.New("Failed to marshal all policies. Check error for details."), errDetails)
	}
}

func writeErrorWithDetails(w http.ResponseWriter, code codes.Code, err error, details ...protoadapt.MessageV1) {
	userErr := status.New(code, err.Error())
	statusMsg, err := userErr.WithDetails(details...)
	if err != nil {
		utils.Should(err)
		httputil.WriteGRPCStyleError(w, codes.Internal, err)
		return
	}
	w.WriteHeader(runtime.HTTPStatusFromCode(code))
	_ = jsonutil.Marshal(w, statusMsg.Proto())
}

func subDNSDomainToZipFileName(name, id string) string {
	if len(name) > maxCustomResourceNameToFilePathTruncSize {
		name = strings.Trim(name[:maxCustomResourceNameToFilePathTruncSize], "-.")
	}
	return name + "-" + id
}
