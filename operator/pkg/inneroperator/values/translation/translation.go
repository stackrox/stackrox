package translation

import (
	"context"

	// Required for the usage of go:embed below.
	_ "embed"

	helmUtil "github.com/stackrox/rox/pkg/helm/util"
	"github.com/stackrox/rox/pkg/utils"
	"helm.sh/helm/v3/pkg/chartutil"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	//go:embed base-values.yaml
	baseValuesYAML []byte
)

// New creates a new Translator
func New(client ctrlClient.Client) Translator {
	return Translator{client: client}
}

// Translator translates and enriches helm values
type Translator struct {
	client ctrlClient.Client
}

// Translate translates and enriches helm values
func (t Translator) Translate(_ context.Context, _ *unstructured.Unstructured) (chartutil.Values, error) {
	baseValues, err := chartutil.ReadValues(baseValuesYAML)
	utils.CrashOnError(err) // ensured through unit test that this doesn't happen.

	// FIXME: Add option to read / overwrite via env vars
	return helmUtil.CoalesceTables(baseValues), nil
}
