import queryString from 'qs';

export function parseFragment(location: Location): Record<string, string> {
    // ignore '#' https://github.com/ljharb/qs/issues/222
    // The fragment as a whole is URL-encoded, which means that each individual field is doubly URL-encoded. We need
    // to decode one additional level of URL encoding here.
    const hash = queryString.parse(location.hash.slice(1));

    const transformedHash: Record<string, string> = {};
    Object.entries(hash).forEach(([key, value]) => {
        transformedHash[key] = decodeURIComponent(value as string);
    });
    return transformedHash;
}
