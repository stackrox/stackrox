/* eslint-env node */
module.exports = {
    root: true,
    env: {
        browser: true,
        es2021: true,
    },
    globals: {
        __ENV: true,
    },
    extends: ['eslint:recommended', 'prettier'],
    overrides: [],
    parserOptions: {
        ecmaVersion: 'latest',
        sourceType: 'module',
    },
    rules: {},
};
