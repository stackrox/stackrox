const proxy = require('http-proxy-middleware');

module.exports = function main(app) {
    const defaultOptions = { target: 'https://localhost:8000', changeOrigin: true, secure: false };

    app.use(proxy('/v1', defaultOptions));
    app.use(proxy('/api', defaultOptions));
    app.use(proxy('/sso', { ...defaultOptions, xfwd: true }));
};
