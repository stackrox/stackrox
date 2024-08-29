package client

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/log"

	v1 "github.com/stackrox/rox/generated/api/v1"
	storage "github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/roxctl/common"
	"github.com/stackrox/rox/roxctl/common/auth"
	roxctlIO "github.com/stackrox/rox/roxctl/common/io"
	"github.com/stackrox/rox/roxctl/common/logger"
	"github.com/stackrox/rox/roxctl/common/printer"
)

//go:generate mockgen-wrapper
type CachedPolicyClient interface {
	List(ctx context.Context) ([]*storage.Policy, error)
	Get(ctx context.Context, name string) (*storage.Policy, bool, error)
	Create(ctx context.Context, policy *storage.Policy) (*storage.Policy, error)
	Update(ctx context.Context, policy *storage.Policy) error
	FlushCache(ctx context.Context) error
	EnsureFresh(ctx context.Context) error
}

//go:generate mockgen-wrapper
type PolicyClient interface {
	List(context.Context) ([]*storage.ListPolicy, error)
	Get(ctx context.Context, id string) (*storage.Policy, error)
	Post(context.Context, *storage.Policy) (*storage.Policy, error)
	Put(context.Context, *storage.Policy) error
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

func (gc *grpcClient) List(ctx context.Context) ([]*storage.ListPolicy, error) {
	allPolicies, err := gc.svc.ListPolicies(ctx, &v1.RawQuery{})
	if err != nil {
		return []*storage.ListPolicy{}, errors.Wrap(err, "Failed to list policies from grpc client")
	}

	return allPolicies.Policies, err
}

func (gc *grpcClient) Get(ctx context.Context, id string) (*storage.Policy, error) {
	policy, err := gc.svc.GetPolicy(ctx, &v1.ResourceByID{Id: id})
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to fetch policy %s", id)
	}

	return policy, nil
}

func (gc *grpcClient) Post(ctx context.Context, policy *storage.Policy) (*storage.Policy, error) {
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

func (gc *grpcClient) Put(ctx context.Context, policy *storage.Policy) error {
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

func (c *client) List(ctx context.Context) ([]*storage.Policy, error) {
	list := make([]*storage.Policy, len(c.cache))
	i := 0
	for _, value := range c.cache {
		list[i] = value
		i++
	}
	return list, nil
}

func (c *client) Get(ctx context.Context, name string) (*storage.Policy, bool, error) {
	policy, exists := c.cache[name]
	return policy, exists, nil
}

func (c *client) Create(ctx context.Context, policy *storage.Policy) (*storage.Policy, error) {
	policy, err := c.svc.Post(ctx, policy)

	if err != nil {
		return nil, errors.Wrap(err, "Failed to POST policy")
	}

	c.cache[policy.Name] = policy

	return policy, nil
}

func (c *client) Update(ctx context.Context, policy *storage.Policy) error {
	err := c.svc.Put(ctx, policy)

	if err != nil {
		return errors.Wrap(err, "Failed to PUT policy")
	}

	c.cache[policy.Name] = policy

	return nil
}

func (c *client) FlushCache(ctx context.Context) error {
	if time.Now().Sub(c.lastUpdated).Seconds() < float64(10) {
		// Don't flush the cache more often than every 10s
		return nil
	}

	rlog := log.FromContext(ctx)
	rlog.Info("Flushing policy cache")

	allPolicies, err := c.svc.List(ctx)
	if err != nil {
		return errors.Wrap(err, "Failed to list policies")
	}

	newCache := make(map[string]*storage.Policy, len(allPolicies))

	for _, listPolicy := range allPolicies {
		policy, err := c.svc.Get(ctx, listPolicy.Id)
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
	if time.Now().Sub(c.lastUpdated).Minutes() > float64(5.0) {
		return c.FlushCache(ctx)
	}
	return nil
}
