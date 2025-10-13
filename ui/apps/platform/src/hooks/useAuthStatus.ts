import { useSelector } from 'react-redux';
import { createStructuredSelector } from 'reselect';

import { selectors } from 'reducers';
import type { ResourceName } from 'types/roleResources';
import type { Access } from 'types/role.proto';
import type { AuthProvider } from 'services/AuthService';

type CurrentUser = {
    authProvider: AuthProvider;
    userAttributes: { key: string; values: string[] }[];
    userId: string;
    userInfo: {
        friendlyName: string;
        permissions: { resourceToAccess: Record<ResourceName, Access> };
        roles: { name: string; resourceToAccess: Record<ResourceName, Access> }[];
        username: string;
    };
};

type UseAuthStatusResponse = {
    currentUser: CurrentUser;
};
type CurrentUserSelector = (state) => CurrentUser;

const stateSelector = createStructuredSelector({
    currentUser: selectors.getCurrentUser as CurrentUserSelector,
});

const useAuthStatus = (): UseAuthStatusResponse => {
    const { currentUser } = useSelector(stateSelector);

    return { currentUser };
};

export default useAuthStatus;
