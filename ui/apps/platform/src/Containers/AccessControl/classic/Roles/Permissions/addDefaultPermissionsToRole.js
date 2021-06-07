import { NO_ACCESS } from 'constants/accessControl';

export default function addDefaultPermissionsToRole(resources, role) {
    const modifiedRole = { ...role };
    const resourceToAccess = { ...role.resourceToAccess };
    resources.forEach((resource) => {
        // if the access value for the resource is not available
        if (!resourceToAccess[resource]) {
            resourceToAccess[resource] = NO_ACCESS;
        }
    });
    modifiedRole.resourceToAccess = resourceToAccess;
    return modifiedRole;
}
