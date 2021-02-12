import { accessControlPathV2 } from 'routePaths'; // import { accessControlPath } from 'routePaths';
import { AccessControlEntityType } from 'constants/entityTypes';

export const accessControlPath = accessControlPathV2; // export { accessControlPath };

export const entityPathSegment: Record<AccessControlEntityType, string> = {
    ACCESS_SCOPE: 'access-scopes',
    AUTH_PROVIDER: 'auth-providers',
    PERMISSION_SET: 'permission-sets',
    ROLE: 'roles',
};

export function getEntityPath(entityType: AccessControlEntityType, entityId = ''): string {
    const entityTypePath = `${accessControlPath}/${entityPathSegment[entityType]}`;
    return entityId ? `${entityTypePath}/${entityId}` : entityTypePath;
}
