import { cloneDeep } from 'lodash';

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
        return this.attributesMap.username || this.userData.userInfo?.username;
    }

    get roles() {
        return this.userData.userInfo?.roles;
    }
}
