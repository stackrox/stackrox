const { createProxyMiddleware } = require('http-proxy-middleware');

function proxyWithTarget(target) {
    return {
        target,
        changeOrigin: true,
        secure: false,
        xfwd: true,
    };
}

/**
 * Allows custom proxy endpoints during development by setting the UI_CUSTOM_PROXIES environment
 * variable to a comma separated list of 'path,target' pairs.
 *
 * @example UI_CUSTOM_PROXIES='/v1/collections,http://localhost:3030,/v1/newapi,http://localhost:9000' npm run start
 * @returns {[string, string][]} An array of [path, target] pairs
 */
function parseCustomProxies() {
    const proxyString = process.env.UI_CUSTOM_PROXIES;
    if (!proxyString) {
        return [];
    }

    const rawValues = proxyString.split(',');
    if (rawValues.length % 2 !== 0) {
        // eslint-disable-next-line no-console
        console.warn(
            'UI_CUSTOM_PROXIES must be set with an even number of comma delimited values',
            'Webpack proxy is ignoring this value'
        );
        return [];
    }
    const proxies = [];
    // Iterate over pairs
    for (let i = 0; i < rawValues.length; i += 2) {
        const path = rawValues[i];
        const target = rawValues[i + 1];
        proxies.push([path, target]);
    }
    return proxies;
}

export function viteProxy() {
    const customProxies = Object.fromEntries(
        parseCustomProxies().map(([path, target]) => [path, proxyWithTarget(target)])
    );

    const proxy = proxyWithTarget(process.env.UI_START_TARGET || 'https://localhost:8000');

    return {
        '/v1': proxy,
        '/v2': proxy,
        '/api': proxy,
        '/docs': proxy,
        '/sso': proxy,
    };
}
