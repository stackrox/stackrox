import qs from 'qs';

import { accessControlPathV2 } from 'routePaths'; // import { accessControlPath } from 'routePaths';
import { AccessControlEntityType } from 'constants/entityTypes';

export const accessControlPath = accessControlPathV2; // export { accessControlPath };

export const entityPathSegment: Record<AccessControlEntityType, string> = {
    ACCESS_SCOPE: 'access-scopes',
    AUTH_PROVIDER: 'auth-providers',
    PERMISSION_SET: 'permission-sets',
    ROLE: 'roles',
};

type AccessControlQueryObject = {
    action?: 'create' | 'update';
    s?: Partial<Record<AccessControlEntityType, string>>;
};

export function getEntityPath(
    entityType: AccessControlEntityType,
    entityId = '',
    queryObject?: AccessControlQueryObject
): string {
    const entityTypePath = `${accessControlPath}/${entityPathSegment[entityType]}`;

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
