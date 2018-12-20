export const getExistingGroupsWithDefault = (existingGroups, authProviderId) => {
    const currentlyExistingGroup = existingGroups[authProviderId];
    if (!currentlyExistingGroup) return [];
    const existingGroupsWithDefault = currentlyExistingGroup
        ? currentlyExistingGroup.rules.slice()
        : [];
    if (!currentlyExistingGroup.defaultRole) return existingGroupsWithDefault;
    existingGroupsWithDefault.push({
        props: {
            authProviderId
        },
        roleName: currentlyExistingGroup.defaultRole
    });
    return existingGroupsWithDefault;
};

export const getGroupsWithDefault = (groups, authProviderId, defaultRole) => {
    const groupsWithDefault = [...groups];
    if (!defaultRole) return groupsWithDefault;
    groupsWithDefault.push({
        props: {
            authProviderId
        },
        roleName: defaultRole
    });
    return groupsWithDefault;
};
