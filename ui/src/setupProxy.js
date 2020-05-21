const { createProxyMiddleware } = require('http-proxy-middleware');

const proxyOptions = {
    target: process.env.YARN_START_TARGET || 'https://localhost:8000',
    changeOrigin: true,
    secure: false,
    xfwd: true,
};

module.exports = function main(app) {
    const proxy = createProxyMiddleware(proxyOptions);
    app.use('/v1', proxy);
    app.use('/api', proxy);
    app.use('/docs', proxy);
    app.use('/sso', proxy);
};
