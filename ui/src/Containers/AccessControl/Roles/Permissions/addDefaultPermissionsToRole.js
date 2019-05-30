import isEmpty from 'lodash/isEmpty';
import { NO_ACCESS } from 'constants/accessControl';

export default function addDefaultPermissionsToRole(resources, role) {
    const modifiedRole = { ...role };
    const resourceToAccess = { ...role.resourceToAccess };
    resources.forEach(resource => {
        // if the access value for the resource is not available
        if (!resourceToAccess[resource]) {
            if (isEmpty(role.resourceToAccess)) {
                // use globalAccess level for this resource if resourceToAccess is empty
                resourceToAccess[resource] = role.globalAccess;
            } else {
                resourceToAccess[resource] = NO_ACCESS;
            }
        }
    });
    modifiedRole.resourceToAccess = resourceToAccess;
    return modifiedRole;
}
