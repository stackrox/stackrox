import { cloneDeep } from 'lodash';

import { READ_ACCESS, READ_WRITE_ACCESS } from 'constants/accessControl';

export default class User {
    constructor(userData) {
        this.userData = cloneDeep(userData);
    }

    get attributesMap() {
        const { userAttributes } = this.userData;
        if (!userAttributes) return {};

        // userAttributes is an array of {key, values} objects
        return userAttributes.reduce((res, { key, values }) => ({ ...res, [key]: values[0] }), {});
    }

    get name() {
        return this.attributesMap.name;
    }

    get email() {
        return this.attributesMap.email;
    }

    get username() {
        return this.userData.userInfo?.username || this.attributesMap.username;
    }

    get roles() {
        return this.userData.userInfo?.roles;
    }

    get permissions() {
        return this.userData.userInfo?.permissions;
    }

    get usedAuthProvider() {
        return this.userData.authProvider;
    }

    get resourceToAccessByRole() {
        const resourceToAccessByRole = {};
        this.roles.forEach(({ name, resourceToAccess }) => {
            Object.keys(resourceToAccess).forEach((resourceName) => {
                if (!resourceToAccessByRole[resourceName]) {
                    resourceToAccessByRole[resourceName] = { read: [], write: [] };
                }
                if (resourceToAccess[resourceName] === READ_ACCESS) {
                    resourceToAccessByRole[resourceName].read.push(name);
                }
                if (resourceToAccess[resourceName] === READ_WRITE_ACCESS) {
                    resourceToAccessByRole[resourceName].read.push(name);
                    resourceToAccessByRole[resourceName].write.push(name);
                }
            });
        });
        return resourceToAccessByRole;
    }
}
