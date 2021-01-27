const commonJavaScriptRules = {
    'prettier/prettier': 'error',

    // Require braces even when block has one statement.
    curly: ['error', 'all'],

    // Forbid use of console in favor of proper error capturing.
    'no-console': 'error',
};

module.exports = {
    parser: 'espree',
    parserOptions: {
        ecmaVersion: 2018, // for object spread
        sourceType: 'module',
    },

    plugins: ['prettier'],

    overrides: [
        {
            files: ['*.js'],
            env: {
                es2017: true, // for globals
                node: true,
            },
            extends: ['eslint:recommended', 'prettier'],
            rules: {
                ...commonJavaScriptRules,
            },
        },
    ],
};
