package authproviders

import (
	"testing"
)

func GetTestProvider(_ testing.TB) *providerImpl {
	return &providerImpl{}
}
