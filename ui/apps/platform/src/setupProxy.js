const { createProxyMiddleware } = require('http-proxy-middleware');

function proxyWithTarget(target) {
    return createProxyMiddleware({
        target,
        changeOrigin: true,
        secure: false,
        xfwd: true,
    });
}

/**
 * Allows custom proxy endpoints during development by setting the YARN_CUSTOM_PROXIES environment
 * variable to a comma separated list of 'path,target' pairs.
 *
 * @example YARN_CUSTOM_PROXIES='/v1/collections,http://localhost:3030,/v1/newapi,http://localhost:9000' yarn start
 * @returns {[string, string][]} An array of [path, target] pairs
 */
function parseCustomProxies() {
    const proxyString = process.env.YARN_CUSTOM_PROXIES;
    if (!proxyString) {
        return [];
    }
    const rawValues = proxyString.split(',');
    if (rawValues.length % 2 !== 0) {
        // eslint-disable-next-line no-console
        console.warn(
            'YARN_CUSTOM_PROXIES must be set with an even number of comma delimited values',
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

module.exports = function main(app) {
    parseCustomProxies().forEach(([path, target]) => app.use(path, proxyWithTarget(target)));

    const proxy = proxyWithTarget(process.env.YARN_START_TARGET || 'https://localhost:8000');
    app.use('/v1', proxy);
    app.use('/v2', proxy);
    app.use('/api', proxy);
    app.use('/docs', proxy);
    app.use('/sso', proxy);
};
