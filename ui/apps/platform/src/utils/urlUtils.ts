import { generatePath } from 'react-router-dom';

import logError from './logError';

export const httpURLPattern = /(https?:\/\/[a-zA-Z0-9]+\.[^\s]{2,})/g;

const urlPattern = new RegExp(
    '^(https?:\\/\\/)?' + // protocol
        '((([a-z\\d]([a-z\\d-]*[a-z\\d])*)\\.)+[a-z]{2,}|' + // domain name
        '((\\d{1,3}\\.){3}\\d{1,3}))' + // OR ip (v4) address
        '(\\:\\d+)?(\\/[-a-z\\d%:_.~+]*)*' + // port and path
        '(\\?[;&a-z\\d%_.~+=-]*)?' + // query string
        '(\\#[-a-z\\d_]*)?$',
    'i'
); // fragment locator

// for IPv4 blocks: https://www.regextester.com/93987
// for IPv6 blocks: https://blog.markhatton.co.uk/2011/03/15/regular-expressions-for-ip-addresses-cidr-ranges-and-hostnames/
const cidrBlockPattern = new RegExp(
    '^([0-9]{1,3}.){3}[0-9]{1,3}(/([0-9]|[1-2][0-9]|3[0-2]))$' + // IPv4 block matcher
        '|' + // or
        '^s*((([0-9A-Fa-f]{1,4}:){7}([0-9A-Fa-f]{1,4}|:))|(([0-9A-Fa-f]{1,4}:){6}(:[0-9A-Fa-f]{1,4}|((25[0-5]|2[0-4]d|1dd|[1-9]?d)(.(25[0-5]|2[0-4]d|1dd|[1-9]?d)){3})|:))|(([0-9A-Fa-f]{1,4}:){5}(((:[0-9A-Fa-f]{1,4}){1,2})|:((25[0-5]|2[0-4]d|1dd|[1-9]?d)(.(25[0-5]|2[0-4]d|1dd|[1-9]?d)){3})|:))|(([0-9A-Fa-f]{1,4}:){4}(((:[0-9A-Fa-f]{1,4}){1,3})|((:[0-9A-Fa-f]{1,4})?:((25[0-5]|2[0-4]d|1dd|[1-9]?d)(.(25[0-5]|2[0-4]d|1dd|[1-9]?d)){3}))|:))|(([0-9A-Fa-f]{1,4}:){3}(((:[0-9A-Fa-f]{1,4}){1,4})|((:[0-9A-Fa-f]{1,4}){0,2}:((25[0-5]|2[0-4]d|1dd|[1-9]?d)(.(25[0-5]|2[0-4]d|1dd|[1-9]?d)){3}))|:))|(([0-9A-Fa-f]{1,4}:){2}(((:[0-9A-Fa-f]{1,4}){1,5})|((:[0-9A-Fa-f]{1,4}){0,3}:((25[0-5]|2[0-4]d|1dd|[1-9]?d)(.(25[0-5]|2[0-4]d|1dd|[1-9]?d)){3}))|:))|(([0-9A-Fa-f]{1,4}:){1}(((:[0-9A-Fa-f]{1,4}){1,6})|((:[0-9A-Fa-f]{1,4}){0,4}:((25[0-5]|2[0-4]d|1dd|[1-9]?d)(.(25[0-5]|2[0-4]d|1dd|[1-9]?d)){3}))|:))|(:(((:[0-9A-Fa-f]{1,4}){1,7})|((:[0-9A-Fa-f]{1,4}){0,5}:((25[0-5]|2[0-4]d|1dd|[1-9]?d)(.(25[0-5]|2[0-4]d|1dd|[1-9]?d)){3}))|:)))(%.+)?s*(/(12[0-8]|1[0-1][0-9]|[1-9][0-9]|[0-9]))$' // IPv6 block matcher
);

export function isValidURL(str) {
    return !!urlPattern.test(str);
}

export function isValidCidrBlock(str) {
    return !!cidrBlockPattern.test(str);
}

/**
 * Try to generate a path from a pattern and object, falling back to a default value if an error occurs.
 * @param pathPattern A path pattern with placeholders for object properties.
 * @param pathObject  An object with properties to replace placeholders in the path pattern.
 * @param fallback  A fallback value to use if an error occurs.
 * @returns A path generated from the pattern and object, or the fallback value if an error occurs.
 */
export function safeGeneratePath(
    pathPattern: string,
    pathObject: Partial<Record<string, unknown>>,
    fallback: string
): string {
    let href: string;

    try {
        href = generatePath(pathPattern, pathObject);
    } catch (error) {
        href = fallback;
        logError(error);
    }

    return href;
}

export default {
    isValidURL,
    isValidCidrBlock,
    safeGeneratePath,
};
