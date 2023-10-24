import cloneDeep from 'lodash/cloneDeep';

import { READ_ACCESS, READ_WRITE_ACCESS } from 'constants/accessControl';

export default class User {
    userData: {
        userAttributes?: [Record<string, string>];
        userInfo?: {
            username?: string;
            friendlyName?: string;
            roles: {
                name: string;
                resourceToAccess: Record<string, string>;
            }[];
            permissions: Record<string, string>[];
        };
        authProvider?: Record<string, string>;
    };

    constructor(userData) {
        this.userData = cloneDeep(userData);
    }

    get attributesMap(): Record<string, string> {
        const { userAttributes } = this.userData;
        if (!userAttributes) {
            return {};
        }

        // userAttributes is an array of {key, values} objects
        return userAttributes.reduce((res, { key, values }) => ({ ...res, [key]: values[0] }), {});
    }

    get name(): string | undefined {
        return this.attributesMap.name || this.username;
    }

    get email() {
        return this.attributesMap.email;
    }

    get username() {
        return (
            (this.userData.userInfo?.username as string) ||
            this.attributesMap.username ||
            this.email
        );
    }

    get displayName() {
        return this.userData.userInfo?.friendlyName || this.name;
    }

    get roles() {
        return this.userData.userInfo?.roles ?? [];
    }

    get permissions() {
        return this.userData.userInfo?.permissions;
    }

    get usedAuthProvider() {
        return this.userData.authProvider;
    }

    get resourceToAccessByRole() {
        const resourceToAccessByRole = {};
        if (Array.isArray(this.roles)) {
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
        }

        return resourceToAccessByRole;
    }
}
