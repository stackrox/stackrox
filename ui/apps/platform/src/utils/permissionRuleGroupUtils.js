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

export const getGroupsWithDefault = (groups, authProviderId, roleName, defaultGroup) => {
    const groupsWithDefault = [...groups];

    // If we have a default group, add it to the list and return.
    if (defaultGroup) {
        groupsWithDefault.push({
            props: {
                authProviderId: defaultGroup.props.authProviderId,
                id: defaultGroup.props.id,
            },
            roleName,
        });
        return groupsWithDefault;
    }
    // The default group is not yet created, meaning we have to make sure we create it here.
    // Set the auth provider ID and role name.
    // Use the default group if we receive a value here.
    groupsWithDefault.push({
        props: {
            authProviderId,
        },
        roleName,
    });
    return groupsWithDefault;
};
