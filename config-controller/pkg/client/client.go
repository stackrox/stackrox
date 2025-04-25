package client

//go:generate mockgen-wrapper CachedCentralClient,CentralClient

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/size"
	"google.golang.org/grpc"
)

var (
	centralHostPort = fmt.Sprintf("central.%s.svc:443", env.Namespace.Setting())
	log             = logging.LoggerForModule()
)

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
	log.Debug("Refreshing Central API token")
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

type CachedCentralClient interface {
	ListPolicies(ctx context.Context) ([]*storage.Policy, error)
	GetPolicy(ctx context.Context, name string) (*storage.Policy, bool, error)
	CreatePolicy(ctx context.Context, policy *storage.Policy) (*storage.Policy, error)
	UpdatePolicy(ctx context.Context, policy *storage.Policy) error
	DeletePolicy(ctx context.Context, name string) error
	GetNotifiers() map[string]string
	GetClusters() map[string]string
	FlushCache(ctx context.Context) error
	EnsureFresh(ctx context.Context) error
}

type CentralClient interface {
	ListPolicies(context.Context) ([]*storage.ListPolicy, error)
	GetPolicy(ctx context.Context, id string) (*storage.Policy, error)
	PostPolicy(context.Context, *storage.Policy) (*storage.Policy, error)
	PutPolicy(context.Context, *storage.Policy) error
	DeletePolicy(ctx context.Context, id string) error
	ListNotifiers(ctx context.Context) ([]*storage.Notifier, error)
	ListClusters(ctx context.Context) ([]*storage.Cluster, error)
	TokenExchange(ctx context.Context) error
}

type grpcClient struct {
	policySvc   v1.PolicyServiceClient
	notifierSvc v1.NotifierServiceClient
	clusterSvc  v1.ClustersServiceClient
	perRPCCreds *perRPCCreds
}

func newGrpcClient(ctx context.Context) (CentralClient, error) {
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
	perRPCCreds.svc = v1.NewAuthServiceClient(conn)

	return &grpcClient{
		perRPCCreds: perRPCCreds,
		policySvc:   v1.NewPolicyServiceClient(conn),
		notifierSvc: v1.NewNotifierServiceClient(conn),
		clusterSvc:  v1.NewClustersServiceClient(conn),
	}, nil
}

func (gc *grpcClient) ListNotifiers(ctx context.Context) ([]*storage.Notifier, error) {
	allNotifiers, err := gc.notifierSvc.GetNotifiers(ctx, &v1.GetNotifiersRequest{})
	if err != nil {
		return []*storage.Notifier{}, errors.Wrap(err, "Failed to list notifiers from grpc client")
	}

	return allNotifiers.Notifiers, nil
}

func (gc *grpcClient) ListClusters(ctx context.Context) ([]*storage.Cluster, error) {
	allClusters, err := gc.clusterSvc.GetClusters(ctx, &v1.GetClustersRequest{})
	if err != nil {
		return []*storage.Cluster{}, errors.Wrap(err, "Failed to list clusters from grpc client")
	}

	return allClusters.Clusters, nil
}

func (gc *grpcClient) ListPolicies(ctx context.Context) ([]*storage.ListPolicy, error) {
	allPolicies, err := gc.policySvc.ListPolicies(ctx, &v1.RawQuery{})
	if err != nil {
		return []*storage.ListPolicy{}, errors.Wrap(err, "Failed to list policies from grpc client")
	}

	return allPolicies.Policies, nil
}

func (gc *grpcClient) GetPolicy(ctx context.Context, id string) (*storage.Policy, error) {
	policy, err := gc.policySvc.GetPolicy(ctx, &v1.ResourceByID{Id: id})
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

	policy, err := gc.policySvc.PostPolicy(ctx, req)

	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create policy %q", policy.GetName())
	}

	return policy, nil
}

func (gc *grpcClient) PutPolicy(ctx context.Context, policy *storage.Policy) error {
	_, err := gc.policySvc.PutPolicy(ctx, policy)

	if err != nil {
		return errors.Wrapf(err, "Failed to update policy %q", policy.GetName())
	}

	return nil
}

func (gc *grpcClient) DeletePolicy(ctx context.Context, id string) error {
	_, err := gc.policySvc.DeletePolicy(ctx, &v1.ResourceByID{Id: id})

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
	centralSvc            CentralClient
	policyObjectCache     map[string]*storage.Policy // policy ID to policy
	policyNameToIDCache   map[string]string          // policy name to policy ID
	notifierNameToIDCache map[string]string          // notifier name to notifier ID
	clusterNameToIDCache  map[string]string          // cluster name to cluster ID
	lastUpdated           time.Time
}

type clientOptions interface {
	Apply(centralClient CachedCentralClient)
}

func New(ctx context.Context, opts ...clientOptions) (CachedCentralClient, error) {
	c := client{}

	for _, o := range opts {
		o.Apply(&c)
	}

	if c.centralSvc == nil {
		gc, err := newGrpcClient(ctx)
		if err != nil {
			log.Error(err, "Failed to connect to Central")
		}

		c.centralSvc = gc
	}

	err := c.EnsureFresh(ctx)
	if err == nil {
		return &c, nil
	}

	// Log the error once and then keep trying silently
	log.Error(err, "Failed to initialize client. Will continue to retry...")

	for {
		if err := c.EnsureFresh(ctx); err != nil {
			log.Warnf("Initialization Error: %s", err)
			time.Sleep(time.Second * 5)
			continue
		}
		break
	}

	return &c, nil
}

func (c *client) GetNotifiers() map[string]string {
	return c.notifierNameToIDCache
}
func (c *client) GetClusters() map[string]string {
	return c.clusterNameToIDCache
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
	log.Infof("Creating policy %q", policy.Name)
	createdPolicy, err := c.centralSvc.PostPolicy(ctx, policy)

	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("Failed to POST policy '%s'", policy.Name))
	}

	c.policyObjectCache[createdPolicy.GetId()] = createdPolicy
	c.policyNameToIDCache[createdPolicy.GetName()] = createdPolicy.GetId()

	return createdPolicy, nil
}

func (c *client) UpdatePolicy(ctx context.Context, policy *storage.Policy) error {
	log.Infof("Updating policy %q", policy.Name)

	var existingPolicyName string
	if id, ok := c.policyNameToIDCache[policy.GetName()]; ok {
		existingPolicyName = c.policyObjectCache[id].GetName()
	}

	// update policy on central
	err := c.centralSvc.PutPolicy(ctx, policy)
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

func (c *client) DeletePolicy(ctx context.Context, policyID string) error {
	log.Infof("Deleting policy %q", policyID)
	policy := c.policyObjectCache[policyID]
	if policy.GetSource() != storage.PolicySource_DECLARATIVE {
		return errors.New(fmt.Sprintf("policy %q is not externally managed and can be deleted only from central", policy.GetName()))
	}

	if err := c.centralSvc.DeletePolicy(ctx, policyID); err != nil {
		return errors.Wrapf(err, "Failed to DELETE policy %q in central", policy.GetName())
	}
	delete(c.policyObjectCache, policyID)
	delete(c.policyNameToIDCache, policy.GetName())
	return nil
}

func (c *client) FlushCache(ctx context.Context) error {
	if time.Since(c.lastUpdated).Seconds() < 1 {
		// Don't flush the cache more often than every 1s
		return nil
	}

	log.Info("Flushing caches")

	log.Debug("Listing policies")
	allPolicies, err := c.centralSvc.ListPolicies(ctx)
	if err != nil {
		return errors.Wrap(err, "Failed to list policies")
	}

	allNotifiers, err := c.centralSvc.ListNotifiers(ctx)
	if err != nil {
		return errors.Wrap(err, "Failed to list notifiers")
	}

	allClusters, err := c.centralSvc.ListClusters(ctx)
	if err != nil {
		return errors.Wrap(err, "Faield to list clusters")
	}

	newPolicyObjectCache := make(map[string]*storage.Policy, len(allPolicies))
	newPolicyNameToIDCache := make(map[string]string, len(allPolicies))
	newClusterNameToIDCache := make(map[string]string, len(allClusters))
	newNotifierNameToIDCache := make(map[string]string, len(allNotifiers))

	for _, listPolicy := range allPolicies {
		log.Debugf("Get policy: %s", listPolicy.GetName())
		policy, err := c.centralSvc.GetPolicy(ctx, listPolicy.Id)
		if err != nil {
			return errors.Wrapf(err, "Failed to fetch policy %s", listPolicy.Id)
		}
		newPolicyObjectCache[policy.GetId()] = policy
		newPolicyNameToIDCache[policy.GetName()] = policy.GetId()
	}

	for _, notifier := range allNotifiers {
		newNotifierNameToIDCache[notifier.GetName()] = notifier.GetId()
	}

	for _, cluster := range allClusters {
		newClusterNameToIDCache[cluster.GetName()] = cluster.GetId()
	}

	c.policyObjectCache = newPolicyObjectCache
	c.policyNameToIDCache = newPolicyNameToIDCache
	c.clusterNameToIDCache = newClusterNameToIDCache
	c.notifierNameToIDCache = newNotifierNameToIDCache

	c.lastUpdated = time.Now()

	return nil
}

func (c *client) EnsureFresh(ctx context.Context) error {
	// Make sure token isn't expired before flushing cache
	if err := c.centralSvc.TokenExchange(ctx); err != nil {
		return err
	}

	if time.Since(c.lastUpdated).Minutes() > 5.0 {
		return c.FlushCache(ctx)
	}

	return nil
}
