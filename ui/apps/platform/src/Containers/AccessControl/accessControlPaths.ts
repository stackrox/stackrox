import { History } from 'react-router-dom';
import qs from 'qs';

import { accessControlBasePathV2, accessControlPathV2 } from 'routePaths'; // import { accessControlPath } from 'routePaths';
import { AccessControlEntityType } from 'constants/entityTypes';

import { AccessControlQueryObject } from './accessControlTypes';

export const accessControlBasePath = accessControlBasePathV2; // export { accessControlBasePath };
export const accessControlPath = accessControlPathV2; // export { accessControlPath };

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
    return qs.parse(search, { ignoreQueryPrefix: true });
}
