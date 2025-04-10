package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"testing"

	notifierStoreMocks "github.com/stackrox/rox/central/notifier/datastore/mocks"
	"github.com/stackrox/rox/central/policy/customresource"
	policyStoreMocks "github.com/stackrox/rox/central/policy/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/apiparams"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stackrox/rox/pkg/zip"
	"github.com/stretchr/testify/suite"
	"github.com/tidwall/gjson"
	gomock "go.uber.org/mock/gomock"
	"golang.org/x/exp/maps"
)

var (
	emailNotifierUuid = uuid.NewV4().String()
	jiraNotifierUuid  = uuid.NewV4().String()
)

func TestPolicyHTTPTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(PolicyHandlerTestSuite))
}

type PolicyHandlerTestSuite struct {
	suite.Suite

	ctx           context.Context
	mockCtrl      *gomock.Controller
	policyStore   *policyStoreMocks.MockDataStore
	notifierStore *notifierStoreMocks.MockDataStore
	handler       http.Handler
	mockRecorder  *httptest.ResponseRecorder
}

func (s *PolicyHandlerTestSuite) SetupTest() {
	s.ctx = context.Background()
	s.mockCtrl = gomock.NewController(s.T())
	s.policyStore = policyStoreMocks.NewMockDataStore(s.mockCtrl)
	s.notifierStore = notifierStoreMocks.NewMockDataStore(s.mockCtrl)
	s.handler = Handler(s.policyStore, s.notifierStore)
	s.mockRecorder = httptest.NewRecorder()
}

func (s *PolicyHandlerTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

// Helper to simulate HTTP POST request
func (s *PolicyHandlerTestSuite) performRequest(body interface{}) *httptest.ResponseRecorder {
	var reqBody []byte
	if body != nil {
		var err error
		reqBody, err = json.Marshal(body)
		s.NoError(err)
	}
	req := httptest.NewRequest("POST", "/url", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	s.handler.ServeHTTP(s.mockRecorder, req)
	return s.mockRecorder
}

// Test when an invalid policy ID is provided
func (s *PolicyHandlerTestSuite) TestSaveAsInvalidIDFails() {
	mockRequest := &apiparams.SaveAsCustomResourcesRequest{IDs: []string{"invalid-id"}}
	mockErrors := []*policyOperationError{
		{
			PolicyId: "invalid-id",
			Error: &v1.PolicyError{
				Error: "not found",
			},
		},
	}

	// Mock GetPolicies call to return no policies
	s.policyStore.EXPECT().GetPolicies(s.ctx, mockRequest.IDs).Return(nil, []int{0}, nil)
	resp := s.performRequest(mockRequest)
	s.Equal(http.StatusBadRequest, resp.Code)
	s.verifyResponse(resp, "Failed to retrieve all policies", http.StatusBadRequest, mockErrors)
}

// Test when a valid policy ID is provided
func (s *PolicyHandlerTestSuite) TestSaveAsValidIDSucceeds() {
	ctx := context.Background()
	expectedNameToPolicies := map[string]*storage.Policy{
		"a-name": {
			Id:   "valid-id",
			Name: "A name",
		},
	}
	mockRequest := &apiparams.SaveAsCustomResourcesRequest{IDs: []string{"valid-id"}}

	s.policyStore.EXPECT().GetPolicies(ctx, mockRequest.IDs).Return(maps.Values(expectedNameToPolicies), nil, nil)
	s.notifierStore.EXPECT().ForEachNotifier(s.ctx, gomock.Any()).DoAndReturn(
		func(_ context.Context, fn func(obj *storage.Notifier) error) error {
			for _, n := range []*storage.Notifier{
				{
					Id:   emailNotifierUuid,
					Name: "email-notifier",
				},
				{
					Id:   jiraNotifierUuid,
					Name: "jira-notifier",
				},
			} {
				if err := fn(n); err != nil {
					return err
				}
			}
			return nil
		})

	resp := s.performRequest(mockRequest)
	s.Equal(http.StatusOK, resp.Code)

	// Expect ZIP file response
	respBody, _ := io.ReadAll(resp.Body)
	s.Contains(string(respBody), "PK") // ZIP files start with "PK" signature
	s.Equal("application/zip", resp.Header().Get("Content-Type"))
	s.Regexp(`attachment; filename="security_policies_.+"`, resp.Header().Get("Content-Disposition"))
	// Create a temp file to save the zip content
	tempZipFile, err := s.dumpToTempFile(respBody)
	s.NoError(err)
	defer func() { _ = os.RemoveAll(tempZipFile) }()
	// Check the contents of the zip file
	s.checkCustomResourcesContents(tempZipFile, expectedNameToPolicies)
}

// Test when multiple valid policy IDs are provided
func (s *PolicyHandlerTestSuite) TestSaveAsMultipleValidIDSucceeds() {
	ctx := context.Background()
	expectedNameToPolicies := map[string]*storage.Policy{
		"policy-1":     {Id: "id1", Name: "Policy 1"},
		"policy-2":     {Id: "id2", Name: "Policy 2"},
		"policy-2-id3": {Id: "id3", Name: "policy 2-"}, // Name conflict
	}
	mockRequest := &apiparams.SaveAsCustomResourcesRequest{IDs: []string{"id1", "id2"}}

	policies := maps.Values(expectedNameToPolicies)
	slices.SortFunc(policies, func(a, b *storage.Policy) int { return strings.Compare(a.GetId(), b.GetId()) })
	s.policyStore.EXPECT().GetPolicies(ctx, mockRequest.IDs).Return(policies, nil, nil)
	s.notifierStore.EXPECT().ForEachNotifier(s.ctx, gomock.Any()).DoAndReturn(
		func(_ context.Context, fn func(obj *storage.Notifier) error) error {
			for _, n := range []*storage.Notifier{
				{
					Id:   emailNotifierUuid,
					Name: "email-notifier",
				},
				{
					Id:   jiraNotifierUuid,
					Name: "jira-notifier",
				},
			} {
				if err := fn(n); err != nil {
					return err
				}
			}
			return nil
		})
	resp := s.performRequest(mockRequest)
	s.Equal(http.StatusOK, resp.Code)

	// Expect ZIP file response
	respBody, _ := io.ReadAll(resp.Body)
	s.Contains(string(respBody), "PK")
	s.Equal("application/zip", resp.Header().Get("Content-Type"))
	s.Regexp(`attachment; filename="security_policies_.+"`, resp.Header().Get("Content-Disposition"))
	// Create a temp file to save the zip content
	tempZipFile, err := s.dumpToTempFile(respBody)
	s.NoError(err)
	defer func() { _ = os.RemoveAll(tempZipFile) }()

	// Check the contents of the zip file
	s.checkCustomResourcesContents(tempZipFile, expectedNameToPolicies)
}

// Test mixed scenario where some policies are found and some are missing
func (s *PolicyHandlerTestSuite) TestSaveAsMixedSuccessAndMissing() {
	ctx := context.Background()
	policies := []*storage.Policy{
		{Id: "id1", Name: "Policy 1"},
	}
	mockRequest := &apiparams.SaveAsCustomResourcesRequest{IDs: []string{"id1", "id2"}}
	mockErrors := []*policyOperationError{
		{
			PolicyId: "id2",
			Error: &v1.PolicyError{
				Error: "not found",
			},
		},
	}

	s.policyStore.EXPECT().GetPolicies(ctx, mockRequest.IDs).Return(policies, []int{1}, nil)
	resp := s.performRequest(mockRequest)
	s.Equal(http.StatusBadRequest, resp.Code)
	s.verifyResponse(resp, "Failed to retrieve all policies", http.StatusBadRequest, mockErrors)
}

// Test when all policy IDs fail
func (s *PolicyHandlerTestSuite) TestSaveAsMultipleFailures() {
	ctx := context.Background()
	mockRequest := &apiparams.SaveAsCustomResourcesRequest{IDs: []string{"id1", "id2"}}
	mockErrors := []*policyOperationError{
		{
			PolicyId: "id1",
			Error: &v1.PolicyError{
				Error: "not found",
			},
		},
		{
			PolicyId: "id2",
			Error: &v1.PolicyError{
				Error: "not found",
			},
		},
	}

	s.policyStore.EXPECT().GetPolicies(ctx, mockRequest.IDs).Return(nil, []int{0, 1}, nil)
	resp := s.performRequest(mockRequest)
	s.Equal(http.StatusBadRequest, resp.Code)

	s.verifyResponse(resp, "Failed to retrieve all policies", http.StatusBadRequest, mockErrors)
}

type policyOperationError struct {
	PolicyId string          `json:"policyId"`
	Error    *v1.PolicyError `json:"error"`
}

func (s *PolicyHandlerTestSuite) verifyResponse(resp *httptest.ResponseRecorder, expectedErrorMsg string, expectedCode int, expectedErrors []*policyOperationError) {
	s.Equal(expectedCode, resp.Code)

	respBody, err := io.ReadAll(resp.Body)
	s.NoError(err)
	var bodyJson map[string]interface{}
	s.NoError(json.Unmarshal(respBody, &bodyJson))

	if expectedErrorMsg != "" {
		errorMessage, ok := bodyJson["message"].(string)
		s.True(ok)
		s.Contains(errorMessage, expectedErrorMsg)
		errorsJson := gjson.Get(string(respBody), "details.0.errors").String()
		s.NotEmpty(errorsJson)
		expected, err := json.Marshal(expectedErrors)
		s.NoError(err)
		s.JSONEq(string(expected), errorsJson)
	}
}

// dumpToTempFile creates a temp file and writes the response body to it.
func (s *PolicyHandlerTestSuite) dumpToTempFile(respBody []byte) (string, error) {
	tempFile, err := os.CreateTemp("", "PolicyHandlerTestSuite")
	s.NoError(err)
	defer utils.IgnoreError(tempFile.Close)
	_, err = tempFile.Write(respBody)
	s.NoError(err)
	return tempFile.Name(), nil
}

// checkCustomResourcesContents reads the zip file and verifies the content.
func (s *PolicyHandlerTestSuite) checkCustomResourcesContents(zipFilePath string, expectedNameToPolicies map[string]*storage.Policy) {
	// Use the zip reader utility to read the zip file
	zipReader, err := zip.NewReader(zipFilePath)
	s.NoError(err)
	defer utils.IgnoreError(zipReader.Close)

	for expectedName, p := range expectedNameToPolicies {
		// Check if the expected file exists in the zip
		fileName := expectedName + ".yaml"
		s.True(zipReader.ContainsFile(fileName), "Expected file %s not found in zip", fileName)
		// Check contents
		fileContent, err := zipReader.ReadFrom(fileName)
		s.NoError(err)
		s.Contains(string(fileContent), p.GetName())
	}
}

func (s *PolicyHandlerTestSuite) TestZipFileName() {
	longNameLength250 := strings.Repeat("0123456789", 21)
	s.Equal(longNameLength250+"012345-id1", subDNSDomainToZipFileName(longNameLength250+"0123456789", "id1"))
	s.Equal(longNameLength250+"012345-id1", subDNSDomainToZipFileName(longNameLength250+"012345678900000000", "id1"))
	s.Equal(longNameLength250+"0123-id1", subDNSDomainToZipFileName(longNameLength250+"0123..6789", "id1"))
	truncatedName := subDNSDomainToZipFileName(longNameLength250+"0123456789", uuid.NewV4().String())
	s.True(strings.HasPrefix(truncatedName, longNameLength250+"012345-"))
	s.Len(truncatedName, customresource.MaxCustomResourceMetadataNameLength)
}
