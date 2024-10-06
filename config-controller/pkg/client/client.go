package client

//go:generate mockgen-wrapper CachedPolicyClient,PolicyClient

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	storage "github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/retry"
	"github.com/stackrox/rox/pkg/size"
	"google.golang.org/grpc"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var centralHostPort = fmt.Sprintf("central.%s.svc:443", env.Namespace.Setting())

type perRPCCreds struct {
	svc         v1.AuthServiceClient
	metadata    map[string]string
	lastUpdated time.Time
}

func (c *perRPCCreds) GetRequestMetadata(_ context.Context, _ ...string) (map[string]string, error) {
	return c.metadata, nil
}

func (c *perRPCCreds) RequireTransportSecurity() bool {
	return true
}

func (c *perRPCCreds) refreshToken(ctx context.Context) error {
	getLogger(ctx).Info("Refreshing Central API token")
	token, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/token")
	if err != nil {
		return errors.WithMessage(err, "error reading service account token file")
	}

	req := v1.ExchangeAuthMachineToMachineTokenRequest{
		IdToken: string(token),
	}

	resp, err := c.svc.ExchangeAuthMachineToMachineToken(ctx, &req)
	if err != nil {
		return errors.Wrap(err, "Failed to exchange token")
	}

	authHeaderValue := fmt.Sprintf("Bearer %s", resp.AccessToken)

	c.metadata = map[string]string{
		"authorization": authHeaderValue,
	}

	c.lastUpdated = time.Now()

	return nil
}

type CachedPolicyClient interface {
	ListPolicies(ctx context.Context) ([]*storage.Policy, error)
	GetPolicy(ctx context.Context, name string) (*storage.Policy, bool, error)
	CreatePolicy(ctx context.Context, policy *storage.Policy) (*storage.Policy, error)
	UpdatePolicy(ctx context.Context, policy *storage.Policy) error
	DeletePolicy(ctx context.Context, name string) error
	FlushCache(ctx context.Context) error
	EnsureFresh(ctx context.Context) error
}

type PolicyClient interface {
	ListPolicies(context.Context) ([]*storage.ListPolicy, error)
	GetPolicy(ctx context.Context, id string) (*storage.Policy, error)
	PostPolicy(context.Context, *storage.Policy) (*storage.Policy, error)
	PutPolicy(context.Context, *storage.Policy) error
	DeletePolicy(ctx context.Context, id string) error
	TokenExchange(ctx context.Context) error
}

type grpcClient struct {
	svc         v1.PolicyServiceClient
	perRPCCreds *perRPCCreds
}

func newGrpcClient(ctx context.Context) (PolicyClient, error) {
	clientconn.SetUserAgent(clientconn.ConfigController)

	dialOpts := []grpc.DialOption{
		grpc.WithNoProxy(),
	}

	perRPCCreds := &perRPCCreds{}
	opts := clientconn.Options{
		InsecureNoTLS:                  false,
		InsecureAllowCredsViaPlaintext: false,
		DialOptions:                    dialOpts,
		PerRPCCreds:                    perRPCCreds,
	}

	callOpts := []grpc.CallOption{grpc.MaxCallRecvMsgSize(12 * size.MB)}

	conn, err := clientconn.GRPCConnection(ctx, mtls.CentralSubject, centralHostPort, opts, grpc.WithDefaultCallOptions(callOpts...))

	if err != nil {
		return nil, errors.Wrap(err, "Failed to create gRPC connection")
	}

	svc := v1.NewPolicyServiceClient(conn)
	perRPCCreds.svc = v1.NewAuthServiceClient(conn)

	return &grpcClient{
		perRPCCreds: perRPCCreds,
		svc:         svc,
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
		return nil, errors.Wrapf(err, "Failed to create policy %q", policy.GetName())
	}

	return policy, nil
}

func (gc *grpcClient) PutPolicy(ctx context.Context, policy *storage.Policy) error {
	_, err := gc.svc.PutPolicy(ctx, policy)

	if err != nil {
		return errors.Wrapf(err, "Failed to update policy %q", policy.GetName())
	}

	return nil
}

func (gc *grpcClient) DeletePolicy(ctx context.Context, id string) error {
	_, err := gc.svc.DeletePolicy(ctx, &v1.ResourceByID{Id: id})

	if err != nil {
		return errors.Wrapf(err, "Failed to delete policy: %s", id)
	}

	return nil
}

func (gc *grpcClient) TokenExchange(ctx context.Context) error {
	if time.Since(gc.perRPCCreds.lastUpdated).Seconds() > 60.0 {
		return gc.perRPCCreds.refreshToken(ctx)
	}
	return nil
}

type client struct {
	svc                 PolicyClient
	policyObjectCache   map[string]*storage.Policy // policy ID to policy
	policyNameToIDCache map[string]string          // policy name to policy ID
	lastUpdated         time.Time
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
		err := retry.WithRetry(func() error {
			gc, innerErr := newGrpcClient(ctx)
			if innerErr != nil {
				getLogger(ctx).Error(innerErr, "Failed to connect to Central")
			}

			c.svc = gc

			if innerErr = c.EnsureFresh(ctx); innerErr != nil {
				getLogger(ctx).Error(innerErr, "Failed to initialize client")
			}

			return innerErr
		}, retry.Tries(10), retry.WithExponentialBackoff())

		if err != nil {
			return nil, errors.Wrap(err, "could not initialize policy client")
		}
	} else {
		if err := c.EnsureFresh(ctx); err != nil {
			getLogger(ctx).Error(err, "Failed to initialize client")
		}
	}

	return &c, nil
}

func (c *client) ListPolicies(_ context.Context) ([]*storage.Policy, error) {
	policies := make([]*storage.Policy, 0, len(c.policyObjectCache))
	for _, value := range c.policyObjectCache {
		policies = append(policies, value)
	}
	return policies, nil
}

func (c *client) GetPolicy(_ context.Context, name string) (*storage.Policy, bool, error) {
	id, exists := c.policyNameToIDCache[name]
	return c.policyObjectCache[id], exists, nil
}

func (c *client) CreatePolicy(ctx context.Context, policy *storage.Policy) (*storage.Policy, error) {
	getLogger(ctx).Info("POST", "policyName", policy.Name)
	createdPolicy, err := c.svc.PostPolicy(ctx, policy)

	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("Failed to POST policy '%s'", policy.Name))
	}

	c.policyObjectCache[createdPolicy.GetId()] = createdPolicy
	c.policyNameToIDCache[createdPolicy.GetName()] = createdPolicy.GetId()

	return createdPolicy, nil
}

func (c *client) UpdatePolicy(ctx context.Context, policy *storage.Policy) error {
	getLogger(ctx).Info("PUT", "policyName", policy.Name)

	var existingPolicyName string
	if id, ok := c.policyNameToIDCache[policy.GetName()]; ok {
		existingPolicyName = c.policyObjectCache[id].GetName()
	}

	// update policy on central
	err := c.svc.PutPolicy(ctx, policy)
	if err != nil {
		return errors.Wrap(err, "Failed to PUT policy")
	}
	// update caches, taking care of the legit rename a declarative policy case
	c.policyObjectCache[policy.GetId()] = policy
	if existingPolicyName != policy.GetName() {
		delete(c.policyNameToIDCache, existingPolicyName)
	}
	c.policyNameToIDCache[policy.GetName()] = policy.GetId()
	return nil
}

func (c *client) DeletePolicy(ctx context.Context, name string) error {
	getLogger(ctx).Info("DELETE", "policyName", name)
	policyID, ok := c.policyNameToIDCache[name]
	if !ok {
		return nil
	}
	policy := c.policyObjectCache[policyID]
	if policy.GetSource() != storage.PolicySource_DECLARATIVE {
		return errors.New(fmt.Sprintf("policy %q is not externally managed and can be deleted only from central", name))
	}

	if err := c.svc.DeletePolicy(ctx, policyID); err != nil {
		return errors.Wrapf(err, "Failed to DELETE policy %q in central", name)
	}
	delete(c.policyObjectCache, policyID)
	delete(c.policyNameToIDCache, policy.GetName())
	return nil
}

func (c *client) FlushCache(ctx context.Context) error {
	if time.Since(c.lastUpdated).Seconds() < 10 {
		// Don't flush the cache more often than every 10s
		return nil
	}

	getLogger(ctx).Info("Flushing policy cache")

	getLogger(ctx).Info("LIST")
	allPolicies, err := c.svc.ListPolicies(ctx)
	if err != nil {
		return errors.Wrap(err, "Failed to list policies")
	}

	newPolicyObjectCache := make(map[string]*storage.Policy, len(allPolicies))
	newPolicyNameToIDCache := make(map[string]string, len(allPolicies))

	for _, listPolicy := range allPolicies {
		getLogger(ctx).Info("GET", "Name", listPolicy.GetName())
		policy, err := c.svc.GetPolicy(ctx, listPolicy.Id)
		if err != nil {
			return errors.Wrapf(err, "Failed to fetch policy %s", listPolicy.Id)
		}
		newPolicyObjectCache[policy.GetId()] = policy
		newPolicyNameToIDCache[policy.GetName()] = policy.GetId()
	}
	c.policyObjectCache = newPolicyObjectCache
	c.policyNameToIDCache = newPolicyNameToIDCache
	c.lastUpdated = time.Now()

	return nil
}

func (c *client) EnsureFresh(ctx context.Context) error {
	// Make sure token isn't expired before flushing cache
	if err := c.svc.TokenExchange(ctx); err != nil {
		return err
	}

	if time.Since(c.lastUpdated).Minutes() > 5.0 {
		return c.FlushCache(ctx)
	}

	return nil
}

func getLogger(ctx context.Context) logr.Logger {
	return log.FromContext(ctx).WithName("central-client")
}
