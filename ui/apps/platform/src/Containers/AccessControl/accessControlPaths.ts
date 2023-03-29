import { History } from 'react-router-dom';
import qs from 'qs';

import { accessControlBasePath } from 'routePaths';
import { AccessControlEntityType } from 'constants/entityTypes';
import { BasePageAction, getQueryObject as baseGetQueryObject } from 'utils/queryStringUtils';

export type AccessControlQueryAction = BasePageAction;

export type AccessControlQueryFilter = Partial<Record<AccessControlEntityType, string>>;

export type AccessControlQueryObject = {
    action?: AccessControlQueryAction;
    s?: AccessControlQueryFilter;
    type?: 'auth0' | 'odic' | 'saml' | 'userpki' | 'iap';
};

export type AccessControlContainerProps = {
    entityId: string;
    history: History;
    queryObject: AccessControlQueryObject;
};

export const entityPathSegment: Record<AccessControlEntityType, string> = {
    AUTH_PROVIDER: 'auth-providers',
    ROLE: 'roles',
    PERMISSION_SET: 'permission-sets',
    ACCESS_SCOPE: 'access-scopes',
};

export function getEntityPath(
    entityType: AccessControlEntityType,
    entityId = '',
    queryObject?: AccessControlQueryObject
): string {
    const entityTypePath = `${accessControlBasePath}/${entityPathSegment[entityType]}`;

    // TODO verify which the backend will expect:
    // ?s[PERMISSION_SET]=GuestAccount
    // ?s[Permission%20Set]=GuestAccount
    const queryString = queryObject
        ? qs.stringify(queryObject, {
              addQueryPrefix: true,
              encodeValuesOnly: true, // TODO for _ but what if space?
          })
        : '';

    return entityId
        ? `${entityTypePath}/${entityId}${queryString}`
        : `${entityTypePath}${queryString}`;
}

export function getQueryObject(search: string): AccessControlQueryObject {
    return baseGetQueryObject(search);
}
