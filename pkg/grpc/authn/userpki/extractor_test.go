package userpki

import (
	"testing"

	"github.com/cloudflare/cfssl/helpers"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	authProviderMocks "github.com/stackrox/rox/pkg/auth/authproviders/mocks"
	"github.com/stackrox/rox/pkg/auth/permissions"
	permissionMocks "github.com/stackrox/rox/pkg/auth/permissions/mocks"
	"github.com/stackrox/rox/pkg/grpc/requestinfo"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/testutils/roletest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/metadata"
)

var userA = []byte(`-----BEGIN CERTIFICATE-----
MIIDQTCCAimgAwIBAgIUTLzG4AllRXc6ongh3HVhXaG649YwDQYJKoZIhvcNAQEL
BQAwOjELMAkGA1UEBhMCVVMxETAPBgNVBAoMCFN0YWNrcm94MRgwFgYDVQQDDA9J
bnRlcm1lZGlhdGUgQ0EwHhcNMTkwNjI0MjIyOTExWhcNMTkwNzI0MjIyOTExWjBB
MQswCQYDVQQGEwJVUzERMA8GA1UECgwIU3RhY2tyb3gxDzANBgNVBAsMBkdyb3Vw
QTEOMAwGA1UEAwwFVXNlckEwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIB
AQDME3pP4HrqIDvFn4/MObdgBpl4cLmZJwyi5V9RMzhG4hIm8Q3B7x5wwgdOPzZT
x7LyUmCMZOOmY1PRnv1kC2LEILoizrO5DuT/aZIv7c8ZHk17ImtJ0Qf5tyQkFnp2
ahnOk8NmfY5fjQbf/yWsF4p9DAywaQzl6DPlIJ5JdC+0KlAsC/INK9mXZlkUY1SQ
oKkJUCG/9J0KMim4shdTaGCcsfD5diIIm9WFt4i8Vpo22TS226e2apLiEoit+DVK
CGd7As+3/fZBR5qoz1SuSYjGYxmDmRGidVdcaEQgCJ0uNAHFrR0TJPYteiDsnhi9
rc0DK8nVwJ20ePuNb+zpYLMJAgMBAAGjODA2MAkGA1UdEwQCMAAwCwYDVR0PBAQD
AgXgMBwGA1UdEQQVMBOBEXVzZXJhQHN0YWNrcm94LmlvMA0GCSqGSIb3DQEBCwUA
A4IBAQBtPFeoExZlv7ql/AiEK5mhRr6ZcvWM7k64CoPQTC8wuh0G/j02qajiArqw
Ex4RonhQaM/+NqpOHOKAQCDjWJedZ7IykMm+V8OucHFiJH/7fEsiNtU0HfJOmb9m
yq3l9TUCE3xQje590zR54kbYnMIMwl+nD8qHuHmXUK6SiosmO4EI9qGI8Rf5VerP
DGXWqqSnyFAH/7RyCpXPRv3Zux9dz+SO5Q59d0fKYzQ6+WsQT0lniqmF4CA2KsTx
F82Ecvn5eGFeGGOrkVdIaQrZzau607vBjG5b3yFZ6FY67D8so4I4b4Bym5k9g12z
f51IDnm9EwnPPJH42AIPiTrsHnLg
-----END CERTIFICATE-----
`)

func TestExtractAttributes(t *testing.T) {
	a := assert.New(t)
	userA, err := helpers.ParseCertificatePEM(userA)
	if err != nil {
		t.Fatal(err)
	}
	ci := requestinfo.ExtractCertInfo(userA)
	values := ExtractAttributes(ci)
	a.ElementsMatch(values[authproviders.EmailAttribute], []string{"usera@stackrox.io"})
	a.ElementsMatch(values[authproviders.GroupsAttribute], []string{"GroupA"})
	a.ElementsMatch(values[authproviders.NameAttribute], []string{"UserA"})

	a.ElementsMatch(values["DN"], []string{"CN=UserA,OU=GroupA,O=Stackrox,C=US"})
}

func TestExtractorIdentityForRequest(t *testing.T) {
	userRequestInfo := requestinfo.RequestInfo{
		Metadata:       metadata.MD{},
		VerifiedChains: [][]mtls.CertInfo{{userCert, caCert}},
	}

	// Requests that yield neither identity nor error.
	// These allow for the next extractor to provide identity information.
	for name, tc := range map[string]struct {
		requestInfo requestinfo.RequestInfo
	}{
		"NoIdentity-NoError - authorization metadata present": {
			// Tokens take precedence over certificates inside the browser app.
			requestInfo: requestinfo.RequestInfo{
				Metadata:       metadata.MD{"authorization": []string{"Bearer dummyTokenData"}},
				VerifiedChains: [][]mtls.CertInfo{{userCert, caCert}},
			},
		},
		"NoIdentity-NoError - no verified chains": {
			requestInfo: requestinfo.RequestInfo{
				Metadata: metadata.MD{},
			},
		},
		"NoIdentity-NoError - no provider for any chain fingerprint": {
			requestInfo: requestinfo.RequestInfo{
				Metadata:       metadata.MD{},
				VerifiedChains: [][]mtls.CertInfo{{userCert, {CertFingerprint: "unknown-fingerprint"}}},
			},
		},
	} {
		t.Run(name, func(it *testing.T) {
			extractor := NewExtractor(&fakeProviderContainer{providers: map[string]authproviders.Provider{}})

			id, err := extractor.IdentityForRequest(it.Context(), tc.requestInfo)
			assert.Nil(it, err)
			assert.Nil(it, id)
		})
	}

	// Error paths.
	t.Run("Error: role resolution fails", func(it *testing.T) {
		mockCtrl := gomock.NewController(it)
		defer mockCtrl.Finish()
		roleMapper := permissionMocks.NewMockRoleMapper(mockCtrl)
		roleMapper.EXPECT().
			FromUserDescriptor(gomock.Any(), gomock.Any()).
			Times(1).
			Return(nil, errDummy)
		mockProvider := authProviderMocks.NewMockProvider(mockCtrl)
		mockProvider.EXPECT().RoleMapper().Times(1).Return(roleMapper)
		extractor := NewExtractor(&fakeProviderContainer{
			providers: map[string]authproviders.Provider{caFingerprint: mockProvider},
		})

		id, err := extractor.IdentityForRequest(it.Context(), userRequestInfo)
		assert.ErrorContains(it, err, "failed to resolve user roles")
		assert.Nil(it, id)
	})

	// Success path.
	t.Run("Valid user certificate", func(it *testing.T) {
		mockCtrl := gomock.NewController(it)
		defer mockCtrl.Finish()
		testRole := roletest.NewResolvedRole("TestRole", testRolePermissions, nil)
		roleMapper := permissionMocks.NewMockRoleMapper(mockCtrl)
		roleMapper.EXPECT().
			FromUserDescriptor(gomock.Any(), &permissions.UserDescriptor{
				UserID:     "userpki:" + userFingerprint,
				Attributes: expectedUserAttributes,
			}).
			Times(1).
			Return([]permissions.ResolvedRole{testRole}, nil)
		mockProvider := authProviderMocks.NewMockProvider(mockCtrl)
		mockProvider.EXPECT().RoleMapper().Times(1).Return(roleMapper)
		extractor := NewExtractor(&fakeProviderContainer{
			providers: map[string]authproviders.Provider{caFingerprint: mockProvider},
		})

		id, err := extractor.IdentityForRequest(it.Context(), userRequestInfo)
		assert.Nil(it, err)
		require.NotNil(it, id)

		assert.Equal(it, "userpki:"+userFingerprint, id.UID())
		assert.Equal(it, "UserA", id.FriendlyName())
		assert.Equal(it, "UserA", id.FullName())
		assert.Equal(it, expectedUserAttributes, id.Attributes())
		assert.Equal(it, []permissions.ResolvedRole{testRole}, id.Roles())
		assert.Equal(it, testRolePermissions, id.Permissions())
		protoassert.Equal(it, &storage.UserInfo{
			FriendlyName: "UserA",
			Permissions:  &storage.UserInfo_ResourceToAccess{ResourceToAccess: testRolePermissions},
			Roles:        []*storage.UserInfo_Role{{Name: "TestRole", ResourceToAccess: testRolePermissions}},
		}, id.User())
		assert.Nil(it, id.Service())
		validFrom, validUntil := id.ValidityPeriod()
		assert.Zero(it, validFrom)
		assert.Equal(it, testNotAfter, validUntil)
		assert.Same(it, mockProvider, id.ExternalAuthProvider())
	})
}
