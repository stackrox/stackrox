import queryString from 'qs';

export function parseFragment(location) {
    const hash = queryString.parse(location.hash.slice(1)); // ignore '#' https://github.com/ljharb/qs/issues/222
    // The fragment as a whole is URL-encoded, which means that each individual field is doubly URL-encoded. We need
    // to decode one additional level of URL encoding here.
    const transformedHash = {};
    Object.entries(hash).forEach(([key, value]) => {
        transformedHash[key] = decodeURIComponent(value);
    });
    return transformedHash;
}
