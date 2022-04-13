package search

import (
	"testing"

	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/alert/convert"
	"github.com/stackrox/stackrox/pkg/fixtures"
	"github.com/stackrox/stackrox/pkg/search"
	"github.com/stretchr/testify/assert"
)

func TestConvertAlert(t *testing.T) {
	nonNamespacedResourceAlert := fixtures.GetResourceAlert()
	nonNamespacedResourceAlert.GetResource().Namespace = ""

	for _, testCase := range []struct {
		desc             string
		alert            *storage.ListAlert
		expectedLocation string
	}{
		{
			desc:             "Deployment alert",
			alert:            convert.AlertToListAlert(fixtures.GetAlert()),
			expectedLocation: "/prod cluster/stackrox/Deployment/nginx_server",
		},
		{
			desc:             "Namespaced resource alert",
			alert:            convert.AlertToListAlert(fixtures.GetResourceAlert()),
			expectedLocation: "/prod cluster/stackrox/Secrets/my-secret",
		},
		{
			desc:             "Non-namespaced resource alert",
			alert:            convert.AlertToListAlert(nonNamespacedResourceAlert),
			expectedLocation: "/prod cluster/Secrets/my-secret",
		},
	} {
		t.Run(testCase.desc, func(t *testing.T) {
			res := convertAlert(testCase.alert, search.Result{})
			assert.Equal(t, testCase.expectedLocation, res.Location)
		})
	}
}
