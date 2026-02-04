package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stackrox/rox/central/baseimage/datastore/repository/mocks"
	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
	delegatorMocks "github.com/stackrox/rox/pkg/delegatedregistry/mocks"
	"github.com/stackrox/rox/pkg/features"
	integrationMocks "github.com/stackrox/rox/pkg/images/integration/mocks"
	"github.com/stackrox/rox/pkg/pointers"
	registryMocks "github.com/stackrox/rox/pkg/registries/mocks"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type ServiceTestSuite struct {
	suite.Suite

	mockCtrl           *gomock.Controller
	mockDatastore      *mocks.MockDataStore
	mockIntegrationSet *integrationMocks.MockSet
	mockRegistrySet    *registryMocks.MockSet
	mockDelegator      *delegatorMocks.MockDelegator
	service            *serviceImpl
}

func TestServiceTestSuite(t *testing.T) {
	suite.Run(t, new(ServiceTestSuite))
}

func (suite *ServiceTestSuite) SetupTest() {
	suite.mockCtrl = gomock.NewController(suite.T())
	suite.mockDatastore = mocks.NewMockDataStore(suite.mockCtrl)
	suite.mockIntegrationSet = integrationMocks.NewMockSet(suite.mockCtrl)
	suite.mockRegistrySet = registryMocks.NewMockSet(suite.mockCtrl)
	suite.mockDelegator = delegatorMocks.NewMockDelegator(suite.mockCtrl)
	suite.service = &serviceImpl{
		datastore:      suite.mockDatastore,
		integrationSet: suite.mockIntegrationSet,
		delegator:      suite.mockDelegator,
	}
}

func (suite *ServiceTestSuite) TearDownTest() {
	suite.mockCtrl.Finish()
}

// setupAllImageRegistriesMatch sets up mocks for registry set matching.
// If matches is true, it expects a matching registry to be found; if false, no match.
func (suite *ServiceTestSuite) setupAllImageRegistriesMatch(matches bool) {
	suite.mockIntegrationSet.EXPECT().RegistrySet().Return(suite.mockRegistrySet)
	suite.mockRegistrySet.EXPECT().Match(gomock.Any()).Return(matches)
}

// setupDelegatorExpectation sets up mocks for delegator behavior.
// If shouldDelegate is true, expects delegation to succeed; if false, expects no delegation.
// This also enables the DelegatedBaseImageScanning feature flag since the delegator is only
// called when the feature is enabled.
func (suite *ServiceTestSuite) setupDelegatorExpectation(shouldDelegate bool) {
	testutils.MustUpdateFeature(suite.T(), features.DelegatedBaseImageScanning, true)
	if shouldDelegate {
		suite.mockDelegator.EXPECT().GetDelegateClusterID(gomock.Any(), gomock.Any()).Return("cluster-1", true, nil)
	} else {
		suite.mockDelegator.EXPECT().GetDelegateClusterID(gomock.Any(), gomock.Any()).Return("", false, nil)
	}
}

func (suite *ServiceTestSuite) TestValidateBaseImageRepository() {
	tests := []struct {
		description         string
		input               string
		expectedValid       bool
		expectedErrMsg      string
		expectRegistryMatch *bool // nil = no match check, true = match, false = no match
		expectDelegation    *bool // nil = no delegation check, true = should delegate, false = should not delegate
	}{
		{
			description:         "accepts simple lowercase repository name",
			input:               "test_com",
			expectedValid:       true,
			expectedErrMsg:      "",
			expectRegistryMatch: pointers.Bool(true),
			expectDelegation:    pointers.Bool(false),
		},
		{
			description:         "accepts standard Docker registry format with organization",
			input:               "docker.io/library/nginx",
			expectedValid:       true,
			expectedErrMsg:      "",
			expectRegistryMatch: pointers.Bool(true),
			expectDelegation:    pointers.Bool(false),
		},
		{
			description:         "accepts IPv4 registry with custom port",
			input:               "192.168.1.1:5000/myapp",
			expectedValid:       true,
			expectedErrMsg:      "",
			expectRegistryMatch: pointers.Bool(true),
			expectDelegation:    pointers.Bool(false),
		},
		{
			description:         "accepts IPv6 registry with repository path",
			input:               "[2001:db8::1]/repo",
			expectedValid:       true,
			expectedErrMsg:      "",
			expectRegistryMatch: pointers.Bool(true),
			expectDelegation:    pointers.Bool(false),
		},
		{
			description:    "rejects empty repository path",
			input:          "",
			expectedValid:  false,
			expectedErrMsg: "invalid base image repository path ''",
		},
		{
			description:    "rejects repository path containing tag",
			input:          "nginx:latest",
			expectedValid:  false,
			expectedErrMsg: "repository path 'nginx:latest' must not include tag - please put tag in the tag pattern field",
		},
		{
			description:    "rejects repository path containing digest",
			input:          "nginx@sha256:abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
			expectedValid:  false,
			expectedErrMsg: "repository path 'nginx@sha256:abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890' must not include digest",
		},
		{
			description:    "rejects repository path with uppercase characters",
			input:          "test:5000/Uppercase/repo",
			expectedValid:  false,
			expectedErrMsg: "invalid base image repository path 'test:5000/Uppercase/repo'",
		},
		{
			description:    "rejects repository path longer than 255 characters",
			input:          strings.Repeat("a", 257),
			expectedValid:  false,
			expectedErrMsg: "invalid base image repository path 'aaaaaaaa",
		},
		{
			description:         "rejects repository path with no matching registry",
			input:               "docker.io/library/nginx",
			expectedValid:       false,
			expectedErrMsg:      "no matching image integration found: please add an image integration for 'docker.io/library/nginx'",
			expectRegistryMatch: pointers.Bool(false),
			expectDelegation:    pointers.Bool(false),
		},
		{
			description:         "accepts repository path with matching central integration",
			input:               "delegated.registry.com/myapp",
			expectedValid:       true,
			expectedErrMsg:      "",
			expectRegistryMatch: pointers.Bool(true),
			expectDelegation:    pointers.Bool(false),
		},
		{
			description:         "rejects repository path when neither central integration nor delegation exists",
			input:               "unknown.registry.com/myapp",
			expectedValid:       false,
			expectedErrMsg:      "no matching image integration found: please add an image integration for 'unknown.registry.com/myapp'",
			expectRegistryMatch: pointers.Bool(false),
			expectDelegation:    pointers.Bool(false),
		},
	}

	for _, tt := range tests {
		suite.Run(tt.description, func() {
			if tt.expectDelegation != nil {
				suite.setupDelegatorExpectation(*tt.expectDelegation)
			}
			if tt.expectRegistryMatch != nil {
				suite.setupAllImageRegistriesMatch(*tt.expectRegistryMatch)
			}
			err := suite.service.validateBaseImageRepository(context.Background(), tt.input)
			if tt.expectedValid {
				suite.NoError(err, "validateBaseImageRepository(%q) expected no error", tt.input)
			} else {
				suite.Errorf(err, "validateBaseImageRepository(%q) expected error", tt.input)
				if tt.expectedErrMsg != "" {
					suite.Contains(err.Error(), tt.expectedErrMsg, "validateBaseImageRepository(%q) error message", tt.input)
				}
			}
		})
	}
}

func (suite *ServiceTestSuite) TestIsValidTagPattern() {
	tests := []struct {
		description    string
		input          string
		expectedValid  bool
		expectedErrMsg string
	}{
		{
			description:    "accepts valid pattern for version matching",
			input:          "8.*",
			expectedValid:  true,
			expectedErrMsg: "",
		},
		{
			description:    "accepts tag pattern with multiple wildcards",
			input:          "v*.*.*",
			expectedValid:  true,
			expectedErrMsg: "",
		},
		{
			description:    "accepts literal tag pattern",
			input:          "latest",
			expectedValid:  true,
			expectedErrMsg: "",
		},
		{
			description:    "accepts wildcard pattern",
			input:          "*",
			expectedValid:  true,
			expectedErrMsg: "",
		},
		{
			description:    "rejects empty tag pattern",
			input:          "",
			expectedValid:  false,
			expectedErrMsg: "tag pattern cannot be empty",
		},
		{
			description:    "accepts tag pattern with question mark",
			input:          "v?.?",
			expectedValid:  true,
			expectedErrMsg: "",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.description, func() {
			valid, err := isValidTagPattern(tt.input)
			suite.Equal(tt.expectedValid, valid, "isValidTagPattern(%q) valid", tt.input)
			if tt.expectedErrMsg != "" {
				suite.Errorf(err, "isValidTagPattern(%q) expected error", tt.input)
				suite.Contains(err.Error(), tt.expectedErrMsg, "isValidTagPattern(%q) error message", tt.input)
			} else {
				suite.NoError(err, "isValidTagPattern(%q) expected no error", tt.input)
			}
		})
	}
}

func (suite *ServiceTestSuite) TestCreateBaseImageReference() {

	tests := []struct {
		description   string
		request       *v2.CreateBaseImageReferenceRequest
		mockSetup     func()
		expectedError bool
		errorContains string
	}{
		{
			description: "creates base image reference with valid inputs",
			request: &v2.CreateBaseImageReferenceRequest{
				BaseImageRepoPath:   "docker.io/library/nginx",
				BaseImageTagPattern: "8.*",
			},
			mockSetup: func() {
				suite.setupAllImageRegistriesMatch(true)
				suite.setupDelegatorExpectation(false)
				created := &storage.BaseImageRepository{
					Id:             "test-id",
					RepositoryPath: "docker.io/library/nginx",
					TagPattern:     "8.*",
				}
				suite.mockDatastore.EXPECT().UpsertRepository(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, repo *storage.BaseImageRepository) (*storage.BaseImageRepository, error) {
						// Verify the service passes the correct initial values
						suite.Equal("docker.io/library/nginx", repo.GetRepositoryPath(), "RepositoryPath should match request")
						suite.Equal("8.*", repo.GetTagPattern(), "TagPattern should match request")
						return created, nil
					})
			},
			expectedError: false,
		},
		{
			description: "rejects creation with no matching registry",
			request: &v2.CreateBaseImageReferenceRequest{
				BaseImageRepoPath:   "docker.io/library/nginx",
				BaseImageTagPattern: "8.*",
			},
			mockSetup: func() {
				suite.setupAllImageRegistriesMatch(false)
				suite.setupDelegatorExpectation(false)
			},
			expectedError: true,
			errorContains: "no matching image integration found",
		},
		{
			description: "rejects creation with invalid tag pattern",
			request: &v2.CreateBaseImageReferenceRequest{
				BaseImageRepoPath:   "docker.io/library/nginx",
				BaseImageTagPattern: "[unclosed",
			},
			mockSetup: func() {
				suite.setupAllImageRegistriesMatch(true)
				suite.setupDelegatorExpectation(false)
			},
			expectedError: true,
			errorContains: "invalid base image tag pattern",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.description, func() {
			tt.mockSetup()

			_, err := suite.service.CreateBaseImageReference(context.Background(), tt.request)

			if tt.expectedError {
				suite.Error(err, "expected error")
				suite.Contains(err.Error(), tt.errorContains, "error message should contain expected text")
			} else {
				suite.NoError(err, "expected no error")
			}
		})
	}
}

func (suite *ServiceTestSuite) TestUpdateBaseImageTagPattern() {

	tests := []struct {
		description   string
		request       *v2.UpdateBaseImageTagPatternRequest
		mockSetup     func()
		expectedError bool
		errorContains string
	}{
		{
			description: "updates tag pattern with valid glob",
			request: &v2.UpdateBaseImageTagPatternRequest{
				Id:                  "test-id",
				BaseImageTagPattern: "8.*",
			},
			mockSetup: func() {
				existing := &storage.BaseImageRepository{
					Id:             "test-id",
					RepositoryPath: "nginx",
					TagPattern:     "latest",
				}
				suite.mockDatastore.EXPECT().GetRepository(gomock.Any(), "test-id").
					Return(existing, true, nil)
				suite.mockDatastore.EXPECT().UpsertRepository(gomock.Any(), gomock.Any()).
					Return(nil, nil)
			},
			expectedError: false,
		},
		{
			description: "rejects update with invalid tag pattern",
			request: &v2.UpdateBaseImageTagPatternRequest{
				Id:                  "test-id",
				BaseImageTagPattern: "[unclosed",
			},
			mockSetup:     func() {},
			expectedError: true,
			errorContains: "invalid base image tag pattern",
		},
		{
			description: "rejects update with empty ID",
			request: &v2.UpdateBaseImageTagPatternRequest{
				Id:                  "",
				BaseImageTagPattern: "latest",
			},
			mockSetup:     func() {},
			expectedError: true,
			errorContains: "base image reference ID is required",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.description, func() {
			tt.mockSetup()

			_, err := suite.service.UpdateBaseImageTagPattern(context.Background(), tt.request)

			if tt.expectedError {
				suite.Error(err, "expected error")
				suite.Contains(err.Error(), tt.errorContains, "error message should contain expected text")
			} else {
				suite.NoError(err, "expected no error")
			}
		})
	}
}

func (suite *ServiceTestSuite) TestGetBaseImageReferences() {

	tests := []struct {
		description   string
		mockSetup     func()
		expectedCount int
		expectedError bool
		errorContains string
	}{
		{
			description: "returns multiple base image references successfully",
			mockSetup: func() {
				repos := []*storage.BaseImageRepository{
					{
						Id:             "id-1",
						RepositoryPath: "docker.io/library/nginx",
						TagPattern:     "8.*",
					},
					{
						Id:             "id-2",
						RepositoryPath: "docker.io/library/alpine",
						TagPattern:     "latest",
					},
				}
				suite.mockDatastore.EXPECT().ListRepositories(gomock.Any()).
					Return(repos, nil)
			},
			expectedCount: 2,
			expectedError: false,
		},
		{
			description: "returns empty list when no repositories exist",
			mockSetup: func() {
				suite.mockDatastore.EXPECT().ListRepositories(gomock.Any()).
					Return([]*storage.BaseImageRepository{}, nil)
			},
			expectedCount: 0,
			expectedError: false,
		},
		{
			description: "handles datastore error",
			mockSetup: func() {
				suite.mockDatastore.EXPECT().ListRepositories(gomock.Any()).
					Return(nil, errors.New("datastore error"))
			},
			expectedCount: 0,
			expectedError: true,
			errorContains: "failed to get base image repositories",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.description, func() {
			tt.mockSetup()

			response, err := suite.service.GetBaseImageReferences(context.Background(), &v2.Empty{})

			if tt.expectedError {
				suite.Error(err, "expected error")
				suite.Contains(err.Error(), tt.errorContains, "error message should contain expected text")
			} else {
				suite.NoError(err, "expected no error")
				suite.NotNil(response, "expected response")
				suite.Len(response.GetBaseImageReferences(), tt.expectedCount, "expected correct number of references")

				// Verify the conversion is correct for non-empty results
				if tt.expectedCount > 0 && response != nil {
					for i, ref := range response.GetBaseImageReferences() {
						suite.NotEmpty(ref.GetId(), "reference %d should have non-empty ID", i)
						suite.NotEmpty(ref.GetBaseImageRepoPath(), "reference %d should have non-empty repository path", i)
					}
				}
			}
		})
	}
}

func (suite *ServiceTestSuite) TestDeleteBaseImageReference() {

	tests := []struct {
		description   string
		request       *v2.DeleteBaseImageReferenceRequest
		mockSetup     func()
		expectedError bool
		errorContains string
	}{
		{
			description: "deletes base image reference successfully",
			request: &v2.DeleteBaseImageReferenceRequest{
				Id: "test-id",
			},
			mockSetup: func() {
				suite.mockDatastore.EXPECT().DeleteRepository(gomock.Any(), "test-id").
					Return(nil)
			},
			expectedError: false,
		},
		{
			description: "rejects deletion with empty ID",
			request: &v2.DeleteBaseImageReferenceRequest{
				Id: "",
			},
			mockSetup:     func() {},
			expectedError: true,
			errorContains: "base image reference ID is required",
		},
		{
			description: "handles datastore error during deletion",
			request: &v2.DeleteBaseImageReferenceRequest{
				Id: "test-id",
			},
			mockSetup: func() {
				suite.mockDatastore.EXPECT().DeleteRepository(gomock.Any(), "test-id").
					Return(errors.New("datastore deletion error"))
			},
			expectedError: true,
			errorContains: "failed to delete base image repository",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.description, func() {
			tt.mockSetup()

			_, err := suite.service.DeleteBaseImageReference(context.Background(), tt.request)

			if tt.expectedError {
				suite.Error(err, "expected error")
				suite.Contains(err.Error(), tt.errorContains, "error message should contain expected text")
			} else {
				suite.NoError(err, "expected no error")
			}
		})
	}
}

func (suite *ServiceTestSuite) TestValidateBaseImageRepository_DelegatedFeatureFlag() {
	tests := []struct {
		description         string
		featureEnabled      bool
		expectDelegatorCall bool
		registryMatch       bool
		expectedError       bool
	}{
		{
			description:         "feature disabled - delegator not called, central registry checked (match)",
			featureEnabled:      false,
			expectDelegatorCall: false,
			registryMatch:       true,
			expectedError:       false,
		},
		{
			description:         "feature disabled - delegator not called, central registry checked (no match)",
			featureEnabled:      false,
			expectDelegatorCall: false,
			registryMatch:       false,
			expectedError:       true,
		},
		{
			description:         "feature enabled - delegator called, delegation succeeds (skips central registry)",
			featureEnabled:      true,
			expectDelegatorCall: true,
			registryMatch:       false, // irrelevant when delegated
			expectedError:       false,
		},
		{
			description:         "feature enabled - delegator called, not delegated, central registry checked",
			featureEnabled:      true,
			expectDelegatorCall: true,
			registryMatch:       true,
			expectedError:       false,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.description, func() {
			testutils.MustUpdateFeature(suite.T(), features.DelegatedBaseImageScanning, tt.featureEnabled)

			if tt.expectDelegatorCall {
				// When delegator is called, return shouldDelegate=true for the "skips central" case.
				if !tt.registryMatch && !tt.expectedError {
					// Delegation succeeds, so skip central registry check.
					suite.mockDelegator.EXPECT().GetDelegateClusterID(gomock.Any(), gomock.Any()).Return("cluster-1", true, nil)
				} else {
					// Not delegated, so central registry check is performed.
					suite.mockDelegator.EXPECT().GetDelegateClusterID(gomock.Any(), gomock.Any()).Return("", false, nil)
					suite.mockIntegrationSet.EXPECT().RegistrySet().Return(suite.mockRegistrySet)
					suite.mockRegistrySet.EXPECT().Match(gomock.Any()).Return(tt.registryMatch)
				}
			} else {
				// Delegator should not be called, central registry is always checked.
				suite.mockIntegrationSet.EXPECT().RegistrySet().Return(suite.mockRegistrySet)
				suite.mockRegistrySet.EXPECT().Match(gomock.Any()).Return(tt.registryMatch)
			}

			err := suite.service.validateBaseImageRepository(context.Background(), "docker.io/library/nginx")

			if tt.expectedError {
				suite.Error(err, "expected error")
			} else {
				suite.NoError(err, "expected no error")
			}
		})
	}
}
