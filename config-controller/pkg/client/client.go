package client

//go:generate mockgen-wrapper CachedPolicyClient,PolicyClient

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	storage "github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/roxctl/common"
	"github.com/stackrox/rox/roxctl/common/auth"
	roxctlIO "github.com/stackrox/rox/roxctl/common/io"
	"github.com/stackrox/rox/roxctl/common/logger"
	"github.com/stackrox/rox/roxctl/common/printer"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type CachedPolicyClient interface {
	ListPolicies(ctx context.Context) ([]*storage.Policy, error)
	GetPolicy(ctx context.Context, name string) (*storage.Policy, bool, error)
	CreatePolicy(ctx context.Context, policy *storage.Policy) (*storage.Policy, error)
	UpdatePolicy(ctx context.Context, policy *storage.Policy) error
	FlushCache(ctx context.Context) error
	EnsureFresh(ctx context.Context) error
}

type PolicyClient interface {
	ListPolicies(context.Context) ([]*storage.ListPolicy, error)
	GetPolicy(ctx context.Context, id string) (*storage.Policy, error)
	PostPolicy(context.Context, *storage.Policy) (*storage.Policy, error)
	PutPolicy(context.Context, *storage.Policy) error
}

type grpcClient struct {
	svc v1.PolicyServiceClient
}

func newGrpcClient(ctx context.Context) (PolicyClient, error) {
	conn, err := common.GetGRPCConnection(auth.TokenAuth(), logger.NewLogger(roxctlIO.DefaultIO(), printer.DefaultColorPrinter()))
	if err != nil {
		return nil, errors.Wrap(err, "could not establish gRPC connection to Central")
	}
	svc := v1.NewPolicyServiceClient(conn)

	return &grpcClient{
		svc: svc,
	}, nil
}

func (gc *grpcClient) ListPolicies(ctx context.Context) ([]*storage.ListPolicy, error) {
	allPolicies, err := gc.svc.ListPolicies(ctx, &v1.RawQuery{})
	if err != nil {
		return []*storage.ListPolicy{}, errors.Wrap(err, "Failed to list policies from grpc client")
	}

	return allPolicies.Policies, err
}

func (gc *grpcClient) GetPolicy(ctx context.Context, id string) (*storage.Policy, error) {
	policy, err := gc.svc.GetPolicy(ctx, &v1.ResourceByID{Id: id})
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to fetch policy %s", id)
	}

	return policy, nil
}

func (gc *grpcClient) PostPolicy(ctx context.Context, policy *storage.Policy) (*storage.Policy, error) {
	req := &v1.PostPolicyRequest{
		Policy:                 policy,
		EnableStrictValidation: true,
	}

	policy, err := gc.svc.PostPolicy(ctx, req)

	if err != nil {
		return nil, errors.Wrap(err, "Failed to POST policy")
	}

	return policy, nil
}

func (gc *grpcClient) PutPolicy(ctx context.Context, policy *storage.Policy) error {
	_, err := gc.svc.PutPolicy(ctx, policy)

	if err != nil {
		return errors.Wrap(err, "Failed to PUT policy")
	}

	return nil
}

type client struct {
	svc         PolicyClient
	cache       map[string]*storage.Policy
	lastUpdated time.Time
}

type clientOptions interface {
	Apply(CachedPolicyClient)
}

func New(ctx context.Context, opts ...clientOptions) (CachedPolicyClient, error) {
	c := client{}

	for _, o := range opts {
		o.Apply(&c)
	}

	if c.svc == nil {
		gc, err := newGrpcClient(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "could not initialize policy client")
		}

		c.svc = gc
	}

	if err := c.FlushCache(ctx); err != nil {
		return nil, errors.Wrap(err, "Failed to initialize cache")
	}

	return &c, nil
}

func (c *client) ListPolicies(ctx context.Context) ([]*storage.Policy, error) {
	policies := make([]*storage.Policy, len(c.cache))
	i := 0
	for _, value := range c.cache {
		policies[i] = value
		i++
	}
	return policies, nil
}

func (c *client) GetPolicy(ctx context.Context, name string) (*storage.Policy, bool, error) {
	policy, exists := c.cache[name]
	return policy, exists, nil
}

func (c *client) CreatePolicy(ctx context.Context, policy *storage.Policy) (*storage.Policy, error) {
	policy, err := c.svc.PostPolicy(ctx, policy)

	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("Failed to POST policy '%s'", policy.Name))
	}

	c.cache[policy.Name] = policy

	return policy, nil
}

func (c *client) UpdatePolicy(ctx context.Context, policy *storage.Policy) error {
	err := c.svc.PutPolicy(ctx, policy)

	if err != nil {
		return errors.Wrap(err, "Failed to PUT policy")
	}

	c.cache[policy.Name] = policy

	return nil
}

func (c *client) FlushCache(ctx context.Context) error {
	if time.Since(c.lastUpdated).Seconds() < float64(10) {
		// Don't flush the cache more often than every 10s
		return nil
	}

	rlog := log.FromContext(ctx)
	rlog.Info("Flushing policy cache")

	allPolicies, err := c.svc.ListPolicies(ctx)
	if err != nil {
		return errors.Wrap(err, "Failed to list policies")
	}

	newCache := make(map[string]*storage.Policy, len(allPolicies))

	for _, listPolicy := range allPolicies {
		policy, err := c.svc.GetPolicy(ctx, listPolicy.Id)
		if err != nil {
			return errors.Wrapf(err, "Failed to fetch policy %s", listPolicy.Id)
		}
		newCache[policy.Name] = policy
	}

	c.cache = newCache
	c.lastUpdated = time.Now()

	return nil
}

func (c *client) EnsureFresh(ctx context.Context) error {
	if time.Since(c.lastUpdated).Minutes() > float64(5.0) {
		return c.FlushCache(ctx)
	}
	return nil
}
