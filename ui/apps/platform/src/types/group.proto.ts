export type Group = {
    // GroupProperties define the properties of a group, applying to users when their properties match.
    // They also uniquely identify the group with the props.id field.
    props: GroupProperties;

    // This is the name of the role that will apply to users in this group.
    roleName: string;
};

// GroupProperties defines the properties of a group.
// Groups apply to users when their properties match. For instance:
//   * If GroupProperties has only an authProviderId, then that group applies
//     to all users logged in with that auth provider.
//   * If GroupProperties in addition has a claim key, then it applies to all
//     users with that auth provider and the claim key, etc.
export type GroupProperties = {
    // Unique identifier for group properties and respectively the group.
    id: string;

    authProviderId: string;
    key: string;
    value: string;
};

export type GetDefaultGroupRequest = {
    authProviderId: string;
    roleName: string;
};

export type GroupBatchUpdateRequest = {
    previousGroups: Group[];
    requiredGroups: Group[];
};
