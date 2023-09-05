const path = require('path');

const commonRules = {
    'prettier/prettier': 'error',

    // Do not require implicit return value.
    'arrow-body-style': 'off', // override eslint-config-airbnb-base

    // Require braces even when block has one statement.
    curly: ['error', 'all'],

    // Forbid use of console in favor of raven-js for error capturing.
    'no-console': 'error',

    // Allow function hoisting.
    // Supersede rule to prevent error 'React' was used before it was defined:
    'no-use-before-define': 'off', // override eslint-config-airbnb-base
    '@typescript-eslint/no-use-before-define': [
        'error',
        {
            functions: false,
        },
    ],

    // override Airbnb's style to add typescript support
    'import/extensions': [
        'error',
        'ignorePackages',
        {
            js: 'never',
            mjs: 'never',
            ts: 'never',
            tsx: 'never',
        },
    ],

    // Allow a single named export from a module
    'import/prefer-default-export': 'off',

    'import/no-extraneous-dependencies': [
        'error',
        {
            devDependencies: [
                path.join(__dirname, '**/*.test.*'),
                path.join(__dirname, 'cypress/**'),
                path.join(__dirname, 'src/setupTests.js'),
                path.join(__dirname, 'src/setupProxy.js'),
                path.join(__dirname, 'src/test-utils/*'),
                path.join(__dirname, '.prettierrc.js'),
                path.join(__dirname, 'postcss.config.js'),
                path.join(__dirname, 'tailwind.config.js'),
            ],
            optionalDependencies: false,
        },
    ],

    'no-restricted-imports': [
        'error',
        {
            paths: [
                {
                    name: 'axios',
                    importNames: ['default'],
                    message:
                        "Please use the axios exported from 'src/services/instance.js' since we've made modifications to it there.",
                },
                {
                    name: 'tippy.js',
                    message:
                        "Existing components 'Tooltip' and 'HoverHint' might cover your use-case. If not, comment the reason of using 'tippy.js' directly.",
                },
                {
                    name: '@tippyjs/react',
                    message:
                        "Consider using existing components like 'Tooltip' and 'HoverHint' instead, comment the reason if you have to use Tippy directly.",
                },
            ],
        },
    ],
};

const commonReactRules = {
    'react/display-name': 'off',
    'react/jsx-props-no-spreading': 'off',
    'react/static-property-placement': ['error', 'static public field'],
    'react/prop-types': [
        'error',
        {
            skipUndeclared: true,
        },
    ],
    // https://github.com/yannickcr/eslint-plugin-react/blob/master/docs/rules/jsx-filename-extension.md
    // allow JSX in .js files
    'react/jsx-filename-extension': [
        'error',
        {
            extensions: ['.js', '.tsx'],
        },
    ],
    // https://github.com/yannickcr/eslint-plugin-react/blob/master/docs/rules/jsx-no-bind.md
    'react/jsx-no-bind': [
        'error',
        {
            ignoreRefs: true,
            allowArrowFunctions: true,
            allowFunctions: true, // override eslint-config-airbnb to allow as alternative to arrow functions
            allowBind: false,
            ignoreDOMComponents: true,
        },
    ],
    // Neither require nor forbid destructuring assignment for props, state, context.
    'react/destructuring-assignment': 'off',
    // forbid 'data-test-id' in preference of 'data-testid'
    'react/forbid-dom-props': [
        'error',
        {
            forbid: ['data-test-id'],
        },
    ],
    'react-hooks/exhaustive-deps': 'warn',

    // DEPRECATED in favor of label-has-associated-control
    // https://github.com/evcohen/eslint-plugin-jsx-a11y/blob/master/docs/rules/label-has-for.md#rule-details
    'jsx-a11y/label-has-for': 'off',

    // Reconfigure for using react-router Link
    'jsx-a11y/anchor-is-valid': [
        'error',
        {
            components: ['Link'],
            specialLink: ['to', 'hrefLeft', 'hrefRight'],
            aspects: ['noHref', 'invalidHref', 'preferButton'],
        },
    ],
    'jsx-a11y/label-has-associated-control': [
        'error',
        {
            assert: 'either',
            depth: 12,
        },
    ],
};

const commonTypeScriptRules = {
    '@typescript-eslint/array-type': [
        'error',
        {
            default: 'array',
            readonly: 'array',
        },
    ],

    /*
     * @typescript-eslint/eslint-plugin 5 removes the following rule from recommended.
     * Simulate future upgrade and delete the override after upgrade to react-scripts 5.
     */
    '@typescript-eslint/explicit-module-boundary-types': 'off',

    /*
     * Turn off rules from recommended-requiring-type-checking because of
     * irrelevant problems when TypeScript modules import from JavaScript modules.
     */
    '@typescript-eslint/no-unsafe-assignment': 'off',
    '@typescript-eslint/no-unsafe-call': 'off',
    '@typescript-eslint/no-unsafe-member-access': 'off',
};

const commonUnitTestRules = {
    'jest/no-focused-tests': 'error',
    'jest/expect-expect': [
        'error',
        {
            assertFunctionNames: ['expect', 'expectSaga'],
        },
    ],
    'testing-library/consistent-data-testid': [
        'error',
        {
            testIdPattern: '^[A-Za-z]+[\\w\\-\\.]*$',
        },
    ],
};

module.exports = {
    plugins: ['prettier'],
    parser: '@typescript-eslint/parser',
    parserOptions: {
        project: './tsconfig.eslint.json',
        tsconfigRootDir: __dirname,
    },
    extends: ['react-app', 'airbnb', 'plugin:react/recommended', 'prettier', 'prettier/react'],
    settings: {
        // in build scripts we use NODE_PATH, so need to configure eslint-plugin-import
        'import/resolver': {
            node: {
                moduleDirectory: ['node_modules', 'src/'],
                extensions: ['.js', '.ts', '.tsx'],
            },
        },
        'import/extensions': ['.js', '.ts', '.tsx'],
    },
    overrides: [
        {
            files: ['*.js'],
            env: {
                node: true,
            },
            rules: {
                ...commonRules,
            },
        },
        {
            files: ['src/**/*.js'],
            env: {
                browser: true,
            },
            rules: {
                ...commonRules,
                ...commonReactRules,
            },
        },
        {
            files: ['**/*.ts', '**/*.tsx'],
            plugins: ['@typescript-eslint', 'prettier'],
            extends: [
                'react-app',
                // 'eslint:recommended',
                // 'plugin:eslint-comments/recommended',
                'plugin:react/recommended',
                'plugin:@typescript-eslint/recommended',
                'plugin:@typescript-eslint/recommended-requiring-type-checking',
                'airbnb-typescript',
                'prettier',
                'prettier/@typescript-eslint',
                'prettier/react',
            ],
            env: {
                browser: true,
            },
            rules: {
                ...commonRules,
                ...commonReactRules,
                ...commonTypeScriptRules,

                // Provide ECMAScript default values instead of defaultProps.
                'react/require-default-props': 'off',
            },
        },
        {
            files: ['src/**/*.test.js'],
            plugins: ['prettier', 'jest'],
            extends: [
                'react-app',
                'airbnb',
                'plugin:testing-library/react',
                'plugin:import/errors',
                'plugin:import/warnings',
                'plugin:import/typescript',
                'plugin:jest/recommended',
                // 'plugin:jest-dom/recommended',
                'plugin:react/recommended',
                'prettier',
                'prettier/react',
            ],
            env: {
                browser: true,
                jest: true,
            },
            rules: {
                ...commonRules,
                ...commonReactRules,
                ...commonUnitTestRules,
            },
        },
        {
            files: ['src/**/*.test.ts', 'src/**/*.test.tsx'],
            plugins: ['prettier', 'jest'],
            extends: [
                'react-app',
                'plugin:testing-library/react',
                'plugin:jest/recommended',
                // 'plugin:jest-dom/recommended',
                'plugin:react/recommended',
                'plugin:@typescript-eslint/recommended',
                'plugin:@typescript-eslint/recommended-requiring-type-checking',
                'airbnb-typescript',
                'prettier',
                'prettier/@typescript-eslint',
                'prettier/react',
            ],
            env: {
                browser: true,
                jest: true,
            },
            rules: {
                ...commonRules,
                ...commonReactRules,
                ...commonTypeScriptRules,
                ...commonUnitTestRules,
            },
        },
        {
            files: ['cypress/**/*'],
            plugins: ['cypress', 'prettier', 'jest'],
            env: {
                browser: true,
                'cypress/globals': true,
            },
            rules: {
                ...commonRules,

                'func-names': 'off', // omit warnings for anonymous functions to skip individual tests
                'jest/no-focused-tests': 'error',
                'no-unused-expressions': 'off', // allows chai-style "expect(x).to.be.true;"
            },
        },
    ],
};
