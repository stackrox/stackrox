package service

import (
	"context"

	"github.com/gogo/protobuf/proto"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/group/datastore"
	"github.com/stackrox/rox/central/group/datastore/serialize"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/sac/resources"
	"google.golang.org/grpc"
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.Access)): {
			"/v1.GroupService/GetGroups",
			"/v1.GroupService/GetGroup",
		},
		user.With(permissions.Modify(resources.Access)): {
			"/v1.GroupService/BatchUpdate",
			"/v1.GroupService/CreateGroup",
			"/v1.GroupService/UpdateGroup",
			"/v1.GroupService/DeleteGroup",
		},
	})
)

type serviceImpl struct {
	v1.UnimplementedGroupServiceServer

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

	var filter func(*storage.Group) bool
	if authProvider != nil || key != nil || value != nil || id != nil {
		filter = func(group *storage.Group) bool {
			if authProvider != nil && *authProvider != group.GetProps().GetAuthProviderId() {
				return false
			}
			if key != nil && *key != group.GetProps().GetKey() {
				return false
			}
			if value != nil && *value != group.GetProps().GetValue() {
				return false
			}
			if id != nil && *id != group.GetProps().GetId() {
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
	group, err := s.groups.Get(ctx, props)
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
		if err := datastore.ValidateGroup(group, true); err != nil {
			return nil, errox.InvalidArgs.CausedBy(err)
		}
	}
	for _, group := range req.GetRequiredGroups() {
		if err := datastore.ValidateGroup(group, false); err != nil {
			return nil, errox.InvalidArgs.CausedBy(err)
		}
	}

	removed, updated, added := diffGroups(req.GetPreviousGroups(), req.GetRequiredGroups())
	if err := s.groups.Mutate(ctx, removed, updated, added, req.GetForce()); err != nil {
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

func (s *serviceImpl) UpdateGroup(ctx context.Context, updateReq *v1.UpdateGroupRequest) (*v1.Empty, error) {
	err := s.groups.Update(ctx, updateReq.GetGroup(), updateReq.GetForce())
	if err != nil {
		return nil, err
	}
	return &v1.Empty{}, nil
}

func (s *serviceImpl) DeleteGroup(ctx context.Context, deleteReq *v1.DeleteGroupRequest) (*v1.Empty, error) {
	err := s.groups.Remove(ctx, &storage.GroupProperties{
		Id:             deleteReq.GetId(),
		AuthProviderId: deleteReq.GetAuthProviderId(),
		Key:            deleteReq.GetKey(),
		Value:          deleteReq.GetValue(),
	}, deleteReq.GetForce())
	if err != nil {
		return nil, err
	}
	return &v1.Empty{}, nil
}

// Helper function that does a diff between two sets of groups and comes up with needed mutations.
func diffGroups(previous []*storage.Group, required []*storage.Group) (removed []*storage.Group, updated []*storage.Group, added []*storage.Group) {
	// This will hold all previous and required groups mapped by their properties and the rolename. It will later on be used
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
		// Ensure we add the required group tuple of role + props to the map. If we wouldn't, it could be possible that
		// an updated group in the required section is a dupe of a newly added group. Instead, we should not create a
		// group if an update to an existing one results in the same tuple of role + props.
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

func dedupeAddedGroups(existingGroupsByPropsAndRole map[string]struct{}, toBeAddedGroups []*storage.Group) []*storage.Group {
	addedGroups := make([]*storage.Group, 0, len(toBeAddedGroups))
	for _, group := range toBeAddedGroups {
		if _, exists := existingGroupsByPropsAndRole[string(serialize.PropsKey(group.GetProps()))+group.GetRoleName()]; !exists {
			// Group does not exist, it can be safely added.
			addedGroups = append(addedGroups, group)
			// Make sure to add the newly props + role name to the map, so we don't mistakenly add the same group twice.
			existingGroupsByPropsAndRole[string(serialize.PropsKey(group.GetProps()))+group.GetRoleName()] = struct{}{}
		}
	}
	return addedGroups
}
