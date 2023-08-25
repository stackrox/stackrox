import encoding from 'k6/encoding';

export function getHeaderWithToken(token) {
    return { authorization: `Bearer ${token}`, 'content-type': 'application/json' };
}

export function getHeaderWithAdminPass(pass) {
    const baseAuth64 = encoding.b64encode(`admin:${pass}`, 'std');

    return { authorization: `Basic ${baseAuth64}`, 'content-type': 'application/json' };
}
