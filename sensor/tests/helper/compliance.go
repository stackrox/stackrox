package helper

import (
	"context"
	"time"

	"github.com/stackrox/rox/compliance/collection/compliance"
	"github.com/stackrox/rox/pkg/retry"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
)

func (c *TestContext) StartCompliance(env *envconf.Config) {
	c.t.Setenv("ROX_ADVERTISED_ENDPOINT", "localhost:443")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	com := compliance.NewComplianceApp(
		&dummyNodeNameProvider{},
		nil,
		retry.NewUnconfirmedMessageHandler(ctx, 5*time.Second))
	com.Start()
}

type dummyNodeNameProvider struct{}

func (dnp *dummyNodeNameProvider) GetNodeName() string {
	return "local-compliance"
}
