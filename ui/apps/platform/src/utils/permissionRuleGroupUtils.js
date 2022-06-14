export const getExistingGroupsWithDefault = (existingGroups, authProviderId) => {
    const currentlyExistingGroup = existingGroups[authProviderId];
    if (!currentlyExistingGroup) {
        return [];
    }
    const existingGroupsWithDefault = currentlyExistingGroup
        ? currentlyExistingGroup.rules.slice()
        : [];
    if (!currentlyExistingGroup.defaultRole) {
        return existingGroupsWithDefault;
    }
    existingGroupsWithDefault.push({
        props: {
            authProviderId,
            id: currentlyExistingGroup.defaultId,
        },
        roleName: currentlyExistingGroup.defaultRole,
    });
    return existingGroupsWithDefault;
};

export const getGroupsWithDefault = (groups, authProviderId, defaultGroup) => {
    const groupsWithDefault = [...groups];
    if (!defaultGroup) {
        return groupsWithDefault;
    }
    groupsWithDefault.push({
        props: {
            authProviderId,
            id: defaultGroup.props.id,
        },
        roleName: defaultGroup.roleName,
    });
    return groupsWithDefault;
};
