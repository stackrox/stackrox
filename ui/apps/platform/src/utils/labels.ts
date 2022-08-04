/*
 * Front-end validation for Kubernetes labels.
 *
 * https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/
 *
 */

const nameRegExp = /^[A-Za-z0-9](?:[-_.A-Za-z0-9]{0,61}[A-Za-z0-9])?$/; // hyphen underscore dot
const subdomainRegExp = /^[A-Za-z0-9](?:[-A-Za-z0-9]{0,61}[A-Za-z0-9])?$/; // hyphen

export function getIsValidLabelKey(key: string): boolean {
    const indexOfSlash = key.indexOf('/');
    const prefix = indexOfSlash === -1 ? '' : key.slice(0, indexOfSlash);
    const name = indexOfSlash === -1 ? key : key.slice(indexOfSlash + 1);

    if (name.length === 0) {
        return false; // name cannot be empty
    }

    if (indexOfSlash !== -1 && prefix.length === 0) {
        return false; // although prefix is optional, it cannot be empty
    }

    if (name.length > 63) {
        return false;
    }

    if (!nameRegExp.test(name)) {
        return false;
    }

    if (prefix.length === 0) {
        return true; // prefix is optional
    }

    if (prefix.length > 253) {
        return false;
    }

    return prefix.split('.').every((subdomain) => subdomainRegExp.test(subdomain));
}

export function getIsValidLabelValue(value: string, isLabelRequired?: boolean): boolean {
    if (value.length === 0) {
        return !isLabelRequired; // value can be empty
    }

    if (value.length > 63) {
        return false;
    }

    return nameRegExp.test(value);
}
