package userpki

import (
	"testing"

	"github.com/cloudflare/cfssl/helpers"
	"github.com/stackrox/stackrox/pkg/auth/authproviders"
	"github.com/stackrox/stackrox/pkg/grpc/requestinfo"
	"github.com/stretchr/testify/assert"
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
