package datastore

import (
	"context"

	"github.com/gogo/protobuf/proto"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	groupFilter "github.com/stackrox/rox/central/group/datastore/filter"
	"github.com/stackrox/rox/central/group/datastore/internal/store"
	"github.com/stackrox/rox/central/role/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/declarativeconfig"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	accessSAC           = sac.ForResource(resources.Access)
	datastoresAccessCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedResourceLevelScopes(
			sac.AccessModeScopeKeyList(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Access)))
)

type dataStoreImpl struct {
	storage               store.Store
	roleDatastore         datastore.DataStore
	authProviderDatastore authproviders.Store

	lock sync.RWMutex
}

func (ds *dataStoreImpl) Upsert(ctx context.Context, group *storage.Group) error {
	if err := sac.VerifyAuthzOK(accessSAC.WriteAllowed(ctx)); err != nil {
		return err
	}

	// Lock to simulate being behind a transaction
	ds.lock.Lock()
	defer ds.lock.Unlock()

	if err := ds.validateAndPrepGroupForUpsertNoLock(ctx, group); err != nil {
		return err
	}

	return wrapAsConflictError(ds.storage.Upsert(ctx, group))
}

func (ds *dataStoreImpl) Get(ctx context.Context, props *storage.GroupProperties) (*storage.Group, error) {
	if ok, err := accessSAC.ReadAllowed(ctx); err != nil {
		return nil, err
	} else if !ok {
		return nil, nil
	}

	if err := ValidateProps(props, true); err != nil {
		return nil, errox.InvalidArgs.CausedBy(err)
	}

	group, exists, err := ds.storage.Get(ctx, props.GetId())
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errox.NotFound.Newf("could not find group %s", props.GetId())
	}
	return group, err
}

func (ds *dataStoreImpl) GetAll(ctx context.Context) ([]*storage.Group, error) {
	if ok, err := accessSAC.ReadAllowed(ctx); err != nil {
		return nil, err
	} else if !ok {
		return nil, nil
	}

	return ds.storage.GetAll(ctx)
}

func (ds *dataStoreImpl) GetFiltered(ctx context.Context, filter func(*storage.Group) bool) ([]*storage.Group, error) {
	if ok, err := accessSAC.ReadAllowed(ctx); err != nil {
		return nil, err
	} else if !ok {
		return nil, nil
	}
	return groupFilter.GetFilteredWithStore(ctx, filter, ds.storage)
}

// Walk is an optimization that allows to search through the datastore and find
// groups that apply to a user within a single transaction.
func (ds *dataStoreImpl) Walk(ctx context.Context, authProviderID string, attributes map[string][]string) ([]*storage.Group, error) {
	if ok, err := accessSAC.ReadAllowed(ctx); err != nil {
		return nil, err
	} else if !ok {
		return nil, nil
	}

	// Search through the datastore and find all groups that apply to a user within a single transaction.
	toSearch := getPossibleGroupProperties(authProviderID, attributes)
	var groups []*storage.Group
	walkFn := func() error {
		groups = groups[:0]
		return ds.storage.Walk(ctx, func(group *storage.Group) error {
			for _, check := range toSearch {
				if propertiesMatch(group.GetProps(), check) {
					groups = append(groups, group)
				}
			}
			return nil
		})
	}
	if err := pgutils.RetryIfPostgres(walkFn); err != nil {
		return nil, err
	}
	return groups, nil
}

func (ds *dataStoreImpl) Add(ctx context.Context, group *storage.Group) error {
	if err := sac.VerifyAuthzOK(accessSAC.WriteAllowed(ctx)); err != nil {
		return err
	}

	// Lock to simulate being behind a transaction
	ds.lock.Lock()
	defer ds.lock.Unlock()

	if err := ds.validateAndPrepGroupForAddNoLock(ctx, group); err != nil {
		return err
	}

	return wrapAsConflictError(ds.storage.Upsert(ctx, group))
}

func (ds *dataStoreImpl) Update(ctx context.Context, group *storage.Group, force bool) error {
	if err := sac.VerifyAuthzOK(accessSAC.WriteAllowed(ctx)); err != nil {
		return err
	}

	// Lock to simulate being behind a transaction
	ds.lock.Lock()
	defer ds.lock.Unlock()

	if err := ds.validateAndPrepGroupForUpdateNoLock(ctx, group, force); err != nil {
		return err
	}
	return wrapAsConflictError(ds.storage.Upsert(ctx, group))
}

func (ds *dataStoreImpl) Mutate(ctx context.Context, remove, update, add []*storage.Group, force bool) error {
	if err := sac.VerifyAuthzOK(accessSAC.WriteAllowed(ctx)); err != nil {
		return err
	}

	// Lock to ensure that all mutations happen as one
	ds.lock.Lock()
	defer ds.lock.Unlock()

	for _, group := range add {
		if err := ds.validateAndPrepGroupForAddNoLock(ctx, group); err != nil {
			return err
		}
	}
	if len(add) > 0 {
		if err := ds.storage.UpsertMany(ctx, add); err != nil {
			return wrapAsConflictError(err)
		}
	}

	for _, group := range update {
		if err := ds.validateAndPrepGroupForUpdateNoLock(ctx, group, force); err != nil {
			return err
		}
	}
	if len(update) > 0 {
		if err := ds.storage.UpsertMany(ctx, update); err != nil {
			return wrapAsConflictError(err)
		}
	}

	var idsToRemove []string
	for _, group := range remove {
		if err := ValidateGroup(group, true); err != nil {
			return errox.InvalidArgs.CausedBy(err)
		}
		groupID, err := ds.validateAndPrepGroupForDeleteNoLock(ctx, group.GetProps(), force)
		if err != nil {
			return err
		}
		idsToRemove = append(idsToRemove, groupID)
	}
	if len(remove) > 0 {
		if err := ds.storage.DeleteMany(ctx, idsToRemove); err != nil {
			return err
		}
	}

	return nil
}

func (ds *dataStoreImpl) Remove(ctx context.Context, props *storage.GroupProperties, force bool) error {
	if err := sac.VerifyAuthzOK(accessSAC.WriteAllowed(ctx)); err != nil {
		return err
	}

	// Lock to ensure synchronization between the get and delete
	ds.lock.Lock()
	defer ds.lock.Unlock()

	groupID, err := ds.validateAndPrepGroupForDeleteNoLock(ctx, props, force)
	if err != nil {
		return err
	}

	return ds.storage.Delete(ctx, groupID)
}

func (ds *dataStoreImpl) RemoveAllWithAuthProviderID(ctx context.Context, authProviderID string, force bool) error {
	groups, err := ds.GetFiltered(ctx, func(group *storage.Group) bool {
		return authProviderID == group.GetProps().GetAuthProviderId()
	})
	if err != nil {
		return errors.Wrap(err, "collecting associated groups")
	}
	return ds.Mutate(ctx, groups, nil, nil, force)
}

func (ds *dataStoreImpl) RemoveAllWithEmptyProperties(ctx context.Context) error {
	// Search through all groups and verify whether any group exists with empty properties and attempt to delete them.
	isEmptyGroupPropertiesF := func(group *storage.Group) bool {
		return group.GetProps().GetAuthProviderId() == "" && group.GetProps().GetKey() == "" &&
			group.GetProps().GetValue() == ""
	}
	groups, err := ds.GetFiltered(ctx, isEmptyGroupPropertiesF)
	if err != nil {
		return err
	}

	var removeGroupErrs *multierror.Error
	for _, group := range groups {
		// Since we are dealing with empty properties, we only require the ID to be set.
		// In case the ID is not set, add the error to the error list.
		id := group.GetProps().GetId()
		if id == "" {
			removeGroupErrs = multierror.Append(removeGroupErrs, errox.InvalidArgs.Newf("group %s has no ID"+
				" set and cannot be deleted", proto.MarshalTextString(group)))
			continue
		}
		if err := ds.storage.Delete(ctx, id); err != nil {
			removeGroupErrs = multierror.Append(removeGroupErrs, err)
		}
	}
	return removeGroupErrs.ErrorOrNil()
}

// Helpers
//////////

// Validate if the group is allowed to be upserted and prep the group before it is upserted in db.
// NOTE: This function assumes that the call to this function is already behind a lock.
func (ds *dataStoreImpl) validateAndPrepGroupForUpsertNoLock(ctx context.Context, newGroup *storage.Group) error {
	if err := ValidateGroup(newGroup, false); err != nil {
		return errox.InvalidArgs.CausedBy(err)
	}

	// Ignoring error on purpose as error is equivalent to ID being already set in this function.
	_ = setGroupIDIfEmpty(newGroup)

	oldGroup, exists, err := ds.storage.Get(ctx, newGroup.GetProps().GetId())
	if err != nil {
		return err
	}
	if exists {
		if err = verifyGroupOrigin(ctx, oldGroup); err != nil {
			return err
		}
	}
	if err = verifyGroupOrigin(ctx, newGroup); err != nil {
		return err
	}

	defaultGroup, err := ds.getDefaultGroupForProps(ctx, newGroup.GetProps())
	if err != nil {
		return err
	}

	// Only disallow update of a default group if it does not update the existing default group, if there is any.
	if defaultGroup != nil && defaultGroup.GetProps().GetId() != newGroup.GetProps().GetId() {
		return errox.AlreadyExists.Newf("cannot update group to default group of auth provider %q as a default group already exists",
			newGroup.GetProps().GetAuthProviderId())
	}
	if err := ds.verifyReferencedRoleAndProvider(newGroup); err != nil {
		return err
	}

	return nil
}

func (ds *dataStoreImpl) verifyReferencedRoleAndProvider(group *storage.Group) error {
	role, found, err := ds.roleDatastore.GetRole(datastoresAccessCtx, group.GetRoleName())
	if err != nil {
		return err
	}
	if !found {
		return errox.InvalidArgs.Newf("group %q role name %q does not exist", group.GetProps().GetId(), group.GetRoleName())
	}
	if err := declarativeconfig.VerifyReferencedResourceOrigin(role, group.GetProps(), role.GetName(), group.GetProps().GetId()); err != nil {
		return err
	}

	authProvider, found, err := ds.authProviderDatastore.GetAuthProvider(datastoresAccessCtx, group.GetProps().GetAuthProviderId())
	if err != nil {
		return err
	}
	if !found {
		return errox.InvalidArgs.Newf("group %q auth provider %q does not exist", group.GetProps().GetId(), group.GetProps().GetAuthProviderId())
	}
	if err := declarativeconfig.VerifyReferencedResourceOrigin(authProvider, group.GetProps(), authProvider.GetName(), group.GetProps().GetId()); err != nil {
		return err
	}
	return nil
}

// Validate if the group is allowed to be added and prep the group before it is added to the db.
// NOTE: This function assumes that the call to this function is already behind a lock.
func (ds *dataStoreImpl) validateAndPrepGroupForAddNoLock(ctx context.Context, group *storage.Group) error {
	if err := ValidateGroup(group, false); err != nil {
		return errox.InvalidArgs.CausedBy(err)
	}

	if err := setGroupIDIfEmpty(group); err != nil {
		return err
	}

	if err := verifyGroupOrigin(ctx, group); err != nil {
		return errors.Wrap(err, "origin didn't match for new group")
	}

	defaultGroup, err := ds.getDefaultGroupForProps(ctx, group.GetProps())
	if err != nil {
		return err
	}

	// Check whether the to-be-added group is a default group, ensure that it does not yet exist.
	if defaultGroup != nil {
		return errox.AlreadyExists.Newf("cannot add a default group of auth provider %q as a default group already exists",
			group.GetProps().GetAuthProviderId())
	}
	if err := ds.verifyReferencedRoleAndProvider(group); err != nil {
		return err
	}

	return nil
}

// Validate if the group is allowed to be updated and prep the group before it is updated in db.
// NOTE: This function assumes that the call to this function is already behind a lock.
func (ds *dataStoreImpl) validateAndPrepGroupForUpdateNoLock(ctx context.Context, group *storage.Group,
	force bool) error {
	if err := ValidateGroup(group, true); err != nil {
		return errox.InvalidArgs.CausedBy(err)
	}

	existingGroup, err := ds.validateMutableGroupIDNoLock(ctx, group.GetProps().GetId(), force)
	if err != nil {
		return err
	}
	if err = verifyGroupOrigin(ctx, existingGroup); err != nil {
		return errors.Wrap(err, "origin didn't match for existing group")
	}
	if err = verifyGroupOrigin(ctx, group); err != nil {
		return errors.Wrap(err, "origin didn't match for new group")
	}

	defaultGroup, err := ds.getDefaultGroupForProps(ctx, group.GetProps())
	if err != nil {
		return err
	}

	// Only disallow update of a default group if it does not update the existing default group, if there is any.
	if defaultGroup != nil && defaultGroup.GetProps().GetId() != group.GetProps().Id {
		return errox.AlreadyExists.Newf("cannot update group to default group of auth provider %q as a default group already exists",
			group.GetProps().GetAuthProviderId())
	}
	if err := ds.verifyReferencedRoleAndProvider(group); err != nil {
		return err
	}
	return nil
}

// Validate the props, fetch the group and check if it is allowed to be deleted.
// NOTE: This function assumes that the call to this function is already behind a lock.
func (ds *dataStoreImpl) validateAndPrepGroupForDeleteNoLock(ctx context.Context, props *storage.GroupProperties,
	force bool) (string, error) {
	if err := ValidateProps(props, true); err != nil {
		return "", errox.InvalidArgs.CausedBy(err)
	}

	propsID := props.GetId()

	group, err := ds.validateMutableGroupIDNoLock(ctx, propsID, force)
	if err != nil {
		return "", err
	}
	if err = verifyGroupOrigin(ctx, group); err != nil {
		return "", err
	}

	return propsID, nil
}

func setGroupIDIfEmpty(group *storage.Group) error {
	if group.GetProps().GetId() != "" {
		return errox.InvalidArgs.Newf("id should be empty but %q was provided", group.GetProps().GetId())
	}
	if group.GetProps() != nil {
		group.GetProps().Id = GenerateGroupID()
	} else {
		// Theoretically should never happen, as the auth provider ID is required to be set.
		group.Props = &storage.GroupProperties{Id: GenerateGroupID()}
	}
	return nil
}

func propertiesMatch(props *storage.GroupProperties, expected *storage.GroupProperties) bool {
	return expected.GetAuthProviderId() == props.GetAuthProviderId() &&
		expected.GetKey() == props.GetKey() &&
		expected.GetValue() == props.GetValue()
}

// When given an auth provider and attributes, we will look for all keys and
// key/value pairs that exist in the datastore for the given auth provider.
func getPossibleGroupProperties(authProviderID string, attributes map[string][]string) (props []*storage.GroupProperties) {
	// Need to consider no key.
	props = append(props, &storage.GroupProperties{AuthProviderId: authProviderID})
	for key, values := range attributes {
		// Need to consider key with no value
		props = append(props, &storage.GroupProperties{AuthProviderId: authProviderID, Key: key})
		// Consider all Key/Value pairs present.
		for _, value := range values {
			props = append(props, &storage.GroupProperties{AuthProviderId: authProviderID, Key: key, Value: value})
		}
	}
	return
}

// isDefaultGroup will check whether the given properties are a default group.
// A default group won't have the key and value fields set, only the auth provider ID field.
func isDefaultGroup(props *storage.GroupProperties) bool {
	return props.GetKey() == "" && props.GetValue() == ""
}

// getByProps returns a group matching the given properties if it exists from the store.
// If more than one group is found matching the properties, an error will be returned.
func (ds *dataStoreImpl) getByProps(ctx context.Context, props *storage.GroupProperties) (*storage.Group, error) {
	groups, err := ds.GetFiltered(ctx, func(g *storage.Group) bool {
		return propertiesMatch(g.GetProps(), props)
	})

	if err != nil {
		return nil, err
	}
	if len(groups) == 0 {
		return nil, nil
	}

	if len(groups) > 1 {
		return nil, errox.InvalidArgs.Newf("multiple groups found for properties (auth provider id=%s, key=%s, "+
			"value=%s), provide an ID to retrieve a group unambiguously",
			props.GetAuthProviderId(), props.GetKey(), props.GetValue())
	}

	return groups[0], nil
}

// getDefaultGroupForProps will check if the given properties are a default group and, if they are, search the
// store for the given auth provider ID, and return the default group if it exists.
// If the properties do not indicate a default group or the default group does not yet exist, it will return nil.
// Otherwise, it will return the default group.
func (ds *dataStoreImpl) getDefaultGroupForProps(ctx context.Context, props *storage.GroupProperties) (*storage.Group, error) {
	// 1. Short-circuit if the props do not indicate a default group. A default group only has the auth provider ID
	// field set.
	if !isDefaultGroup(props) {
		return nil, nil
	}

	// 2. Filter for the default group.
	return ds.getByProps(ctx, &storage.GroupProperties{AuthProviderId: props.GetAuthProviderId()})
}

// validateMutableGroupIDNoLock validates whether a group allows changes or not based on the mutability mode set.
// NOTE: This function assumes that the call to this function is already behind a lock.
func (ds *dataStoreImpl) validateMutableGroupIDNoLock(ctx context.Context, id string, force bool) (*storage.Group, error) {
	group, err := ds.validateGroupExists(ctx, id)
	if err != nil {
		return nil, err
	}

	switch group.GetProps().GetTraits().GetMutabilityMode() {
	case storage.Traits_ALLOW_MUTATE:
		return group, nil
	case storage.Traits_ALLOW_MUTATE_FORCED:
		if force {
			return group, nil
		}
		return nil, errox.InvalidArgs.Newf("group %q is immutable and can only be removed"+
			" via API and specifying the force flag", id)
	default:
		utils.Should(errors.Wrapf(errox.InvalidArgs, "unknown mutability mode given: %q",
			group.GetProps().GetTraits().GetMutabilityMode().String()))
	}
	return nil, errox.InvalidArgs.Newf("group %q is immutable", id)
}

func verifyGroupOrigin(ctx context.Context, group *storage.Group) error {
	if !declarativeconfig.CanModifyResource(ctx, group.GetProps()) {
		return errors.Wrapf(errox.NotAuthorized, "group %q's origin is %s, cannot be modified or deleted with the current permission",
			group.GetProps().GetId(), group.GetProps().GetTraits().GetOrigin())
	}
	return nil
}

func (ds *dataStoreImpl) validateGroupExists(ctx context.Context, id string) (*storage.Group, error) {
	group, exists, err := ds.storage.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errox.NotFound.Newf("group with id %q was not found", id)
	}
	return group, nil
}

// wrapAsConflictError will wrap the error as errox.AlreadyExists if the error indicates a unique
// constraint violation. If not, the error will be returned.
func wrapAsConflictError(err error) error {
	if pgutils.IsUniqueConstraintError(err) {
		return errox.AlreadyExists.CausedBy(err)
	}
	return err
}
