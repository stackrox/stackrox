package service

import (
	"context"

	"github.com/gogo/protobuf/proto"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/group/datastore"
	"github.com/stackrox/rox/central/group/datastore/serialize"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"google.golang.org/grpc"
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.Group)): {
			"/v1.GroupService/GetGroups",
			"/v1.GroupService/GetGroup",
		},
		user.With(permissions.Modify(resources.Group)): {
			"/v1.GroupService/BatchUpdate",
			"/v1.GroupService/CreateGroup",
			"/v1.GroupService/UpdateGroup",
			"/v1.GroupService/DeleteGroup",
		},
	})
)

type serviceImpl struct {
	groups datastore.DataStore
}

func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterGroupServiceServer(grpcServer, s)
}

func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterGroupServiceHandler(ctx, mux, conn)
}

func (*serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

func (s *serviceImpl) GetGroups(ctx context.Context, req *v1.GetGroupsRequest) (*v1.GetGroupsResponse, error) {
	var authProvider, key, value, id *string
	if m, ok := req.GetAuthProviderIdOpt().(*v1.GetGroupsRequest_AuthProviderId); ok {
		authProvider = &m.AuthProviderId
	}
	if m, ok := req.GetKeyOpt().(*v1.GetGroupsRequest_Key); ok {
		key = &m.Key
	}
	if m, ok := req.GetValueOpt().(*v1.GetGroupsRequest_Value); ok {
		value = &m.Value
	}
	if m, ok := req.GetIdOpt().(*v1.GetGroupsRequest_Id); ok {
		id = &m.Id
	}

	var filter func(*storage.GroupProperties) bool
	if authProvider != nil || key != nil || value != nil || id != nil {
		filter = func(props *storage.GroupProperties) bool {
			if authProvider != nil && *authProvider != props.GetAuthProviderId() {
				return false
			}
			if key != nil && *key != props.GetKey() {
				return false
			}
			if value != nil && *value != props.GetValue() {
				return false
			}
			if id != nil && *id != props.GetId() {
				return false
			}
			return true
		}
	}

	groups, err := s.groups.GetFiltered(ctx, filter)
	if err != nil {
		return nil, err
	}

	return &v1.GetGroupsResponse{Groups: groups}, nil
}

func (s *serviceImpl) GetGroup(ctx context.Context, props *storage.GroupProperties) (*storage.Group, error) {
	group, err := s.groups.Get(ctx, props.GetId())
	if err != nil {
		return nil, err
	}
	if group == nil {
		return nil, errors.Wrapf(errox.NotFound, "group %q not found", proto.MarshalTextString(props))
	}
	return group, nil
}

func (s *serviceImpl) BatchUpdate(ctx context.Context, req *v1.GroupBatchUpdateRequest) (*v1.Empty, error) {
	for _, group := range req.GetPreviousGroups() {
		if err := datastore.ValidateGroup(group); err != nil {
			return nil, errors.Wrap(errox.InvalidArgs, err.Error())
		}
	}

	removed, updated, added := diffGroups(req.GetPreviousGroups(), req.GetRequiredGroups())
	if err := s.groups.Mutate(ctx, removed, updated, added); err != nil {
		return nil, err
	}
	return &v1.Empty{}, nil
}

func (s *serviceImpl) CreateGroup(ctx context.Context, group *storage.Group) (*v1.Empty, error) {
	err := s.groups.Add(ctx, group)
	if err != nil {
		return nil, err
	}
	return &v1.Empty{}, nil
}

func (s *serviceImpl) UpdateGroup(ctx context.Context, group *storage.Group) (*v1.Empty, error) {
	err := s.groups.Update(ctx, group)
	if err != nil {
		return nil, err
	}
	return &v1.Empty{}, nil
}

func (s *serviceImpl) DeleteGroup(ctx context.Context, props *storage.GroupProperties) (*v1.Empty, error) {
	if err := datastore.ValidateProps(props); err != nil {
		return nil, err
	}
	err := s.groups.Remove(ctx, props.GetId())
	if err != nil {
		return nil, err
	}
	return &v1.Empty{}, nil
}

// Helper function that does a diff between two sets of groups and comes up with needed mutations.
func diffGroups(previous []*storage.Group, required []*storage.Group) (removed []*storage.Group, updated []*storage.Group, added []*storage.Group) {
	// This will hold all previous groups mapped by their properties and the rolename. It will later on be used
	// to determine whether one of the to-be-created groups is required or already covered by one of the existing
	// groups, iff the properties and role match.
	groupsByPropsAndRole := make(map[string]struct{})

	previousByID := make(map[string]*storage.Group)

	for _, group := range previous {
		previousByID[group.GetProps().GetId()] = group
		groupsByPropsAndRole[string(serialize.PropsKey(group.GetProps()))+group.GetRoleName()] = struct{}{}
	}

	requiredByID := make(map[string]*storage.Group)
	// This will hold all, seemingly to-be-created, groups which do not have an ID set yet.
	requiredWithoutID := make([]*storage.Group, 0, len(required))

	for _, group := range required {
		// Need to deal with groups that do not have an ID yet, since they are potentially new groups.
		if group.GetProps().GetId() == "" {
			requiredWithoutID = append(requiredWithoutID, group)
			continue
		}
		requiredByID[group.GetProps().GetId()] = group
		groupsByPropsAndRole[string(serialize.PropsKey(group.GetProps()))+group.GetRoleName()] = struct{}{}
	}

	for key, group := range previousByID {
		if _, hasRequiredGroup := requiredByID[key]; !hasRequiredGroup {
			removed = append(removed, group)
		}
	}
	for key, group := range requiredByID {
		if previousGroup, hasPreviousGroup := previousByID[key]; hasPreviousGroup {
			if !proto.Equal(previousGroup, group) {
				updated = append(updated, group)
				// Delete the to-be-updated group, otherwise we potentially do not create a group based on stale data.
				delete(groupsByPropsAndRole,
					string(serialize.PropsKey(previousGroup.GetProps()))+previousGroup.GetRoleName())
			}
		}
	}
	added = dedupeAddedGroups(groupsByPropsAndRole, requiredWithoutID)

	return
}

func dedupeAddedGroups(existingGroupsByPropsAndRole map[string]struct{}, added []*storage.Group) []*storage.Group {
	updatedGroups := make([]*storage.Group, 0, len(added))
	for _, group := range added {
		if _, exists := existingGroupsByPropsAndRole[string(serialize.PropsKey(group.GetProps()))+group.GetRoleName()]; !exists {
			// Group does not exist, it can be safely added.
			updatedGroups = append(updatedGroups, group)
			// Make sure to add the newly props + role name to the map, so we don't mistakenly add the same group twice.
			existingGroupsByPropsAndRole[string(serialize.PropsKey(group.GetProps()))+group.GetRoleName()] = struct{}{}
		}
	}
	return updatedGroups
}
