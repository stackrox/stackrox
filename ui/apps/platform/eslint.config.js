'use strict';

const path = require('node:path');

const parserTypeScriptESLint = require('@typescript-eslint/parser');

const pluginAccessibility = require('eslint-plugin-jsx-a11y');
const pluginCypress = require('eslint-plugin-cypress');
const pluginESLint = require('@eslint/js'); // eslint-disable-line import/no-extraneous-dependencies
const pluginESLintComments = require('eslint-plugin-eslint-comments');
const pluginImport = require('eslint-plugin-import');
const pluginJest = require('eslint-plugin-jest');
const pluginJestDOM = require('eslint-plugin-jest-dom');
const pluginPrettier = require('eslint-plugin-prettier');
const pluginReact = require('eslint-plugin-react');
const pluginReactHooks = require('eslint-plugin-react-hooks');
const pluginTestingLibrary = require('eslint-plugin-testing-library');
const pluginTypeScriptESLint = require('@typescript-eslint/eslint-plugin');

const { browser: browserGlobals, jest: jestGlobals, node: nodeGlobals } = require('globals');

const parserAndOptions = {
    parser: parserTypeScriptESLint,
    parserOptions: {
        project: './tsconfig.eslint.json',
        tsconfigRootDir: __dirname,
    },
};

module.exports = [
    {
        // Supersede .eslintignore file.
        // ESLint provides ["**/node_modules/", ".git/"] as default ignores.
        ignores: [
            'build/**',
            'coverage/**',
            'scripts/**',
            'src/setupProxy.js',
            'src/setupTests.js',
        ],
    },
    {
        files: ['**/*.{js,ts,tsx}'], // generic configuration

        // ESLint has cascade for rules (that is, last value for a rule wins).
        // ESLint only replaces other properties (that is, does not spread nor merge).

        // languageOptions are in specific configuration objects

        linterOptions: {
            // reportUnusedDisableDirectives: true, // TODO fix errors
        },

        // Key of plugin is namespace of its rules.
        plugins: {
            'eslint-comments': pluginESLintComments,
            import: pluginImport,
            prettier: pluginPrettier,
        },
        rules: {
            ...pluginESLint.configs.recommended.rules,

            // Require braces even when block has one statement.
            curly: ['error', 'all'],

            // Forbid use of console in favor of raven-js for error capturing.
            'no-console': 'error',

            // ...pluginESLintComments.configs.recommended.rules, // TODO fix errors

            ...pluginImport.configs.errors.rules, // depends on parsers and resolver in settings

            // Turn off rules from import errors configuration.
            'import/named': 'off',

            'import/extensions': [
                'error',
                'ignorePackages',
                {
                    js: 'never',
                    json: 'always',
                    ts: 'never',
                    tsx: 'never',
                },
            ],

            'prettier/prettier': 'error',
        },

        settings: {
            'import/extensions': ['.js', '.ts', '.tsx'],
            'import/parsers': {
                '@typescript-eslint/parser': ['.js', '.ts', '.tsx'],
            },
            'import/resolver': {
                typescript: {
                    alwaysTryTypes: true,
                    project: 'tsconfig.eslint.json',
                },
            },
            react: {
                version: 'detect',
            },
        },
    },
    {
        files: ['cypress/**/*.js'], // helpers (and so on) for integration tests

        languageOptions: {
            ...parserAndOptions,
            globals: {
                ...pluginCypress.environments.globals.globals,
                ...nodeGlobals, // mocha.config.js
            },
        },
    },
    {
        files: ['cypress/integration/**/*.test.js'], // integration tests

        languageOptions: {
            ...parserAndOptions,
            globals: {
                ...pluginCypress.environments.globals.globals,
            },
        },

        // Key of plugin is namespace of its rules.
        plugins: {
            cypress: pluginCypress,
            jest: pluginJest,
        },
        rules: {
            // Turn off rules from ESLint recommended configuration.

            // Omit warnings for anonymous functions to skip individual tests.
            'func-names': 'off',

            // Allow chai-style expect(x).to.be.true chain.
            'no-unused-expressions': 'off',

            // ...pluginCypress.configs.recommended.rules, // TODO fix errors

            // 'cypress/no-force': 'error', // TODO fix errors

            'jest/no-focused-tests': 'error',
        },
    },
    {
        files: ['src/*.{ts,tsx}', 'src/*/**/*.{js,ts,tsx}'], // product files, except for unit tests (including mockData and test-utils folders)

        languageOptions: {
            ...parserAndOptions,
            globals: {
                ...browserGlobals,
                ...nodeGlobals, // TODO limit to minimum used like process.env.WHATEVER
            },
        },

        // Key of plugin is namespace of its rules.
        plugins: {
            import: pluginImport,
            'jsx-a11y': pluginAccessibility,
            react: pluginReact,
            'react-hooks': pluginReactHooks,
        },
        rules: {
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
                    ],
                },
            ],

            'import/no-extraneous-dependencies': [
                'error',
                {
                    devDependencies: [
                        path.join(__dirname, 'src/test-utils/*'), // TODO delete renderWithRedux.js
                    ],
                },
            ],

            // TODO Reconfigure for using react-router Link
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

            ...pluginReact.configs.recommended.rules,

            // Turn off new rules until after we fix errors in follow-up contributions.
            'react/display-name': 'off', // TODO delete after eslint-disable in ReactSelect and URLSearchInputWithAutocomplete
            'react/jsx-key': 'off', // more that 30 errors

            'react/forbid-prop-types': [
                'error',
                {
                    forbid: ['any', 'array'], // allow object
                },
            ],

            'react/jsx-filename-extension': [
                'error',
                {
                    extensions: ['.js', '.tsx'], // allow JSX in .js files
                },
            ],

            'react/jsx-no-bind': [
                'error',
                {
                    allowArrowFunctions: true,
                    allowBind: false,
                    allowFunctions: true,
                    ignoreDOMComponents: true,
                    ignoreRefs: true,
                },
            ],

            'react/prop-types': [
                'error',
                {
                    skipUndeclared: true,
                },
            ],

            'react/static-property-placement': ['error', 'static public field'],

            ...pluginReactHooks.configs.recommended.rules,

            // 'react-hooks/exhaustive-deps': 'warn', // TODO fix errors and then change from default warn to error? or generic warnings as errors?
        },
    },
    {
        files: ['src/**/*.{ts,tsx}'], // product files, except for unit tests (including test-utils folder)

        // languageOptions from previous configuration object

        // Key of plugin is namespace of its rules.
        plugins: {
            '@typescript-eslint': pluginTypeScriptESLint,
        },
        rules: {
            ...pluginTypeScriptESLint.configs['eslint-recommended'].overrides[0].rules,
            ...pluginTypeScriptESLint.configs.recommended.rules,
            ...pluginTypeScriptESLint.configs['recommended-type-checked'].rules,

            // Turn off rules from recommended-type-checked configuration because of
            // irrelevant problems when TypeScript modules import from JavaScript modules.
            '@typescript-eslint/no-unsafe-assignment': 'off',
            '@typescript-eslint/no-unsafe-call': 'off',
            '@typescript-eslint/no-unsafe-member-access': 'off',

            // Turn off new rules until after we fix errors in follow-up contributions.
            '@typescript-eslint/no-duplicate-type-constituents': 'off', // fix 1 error
            '@typescript-eslint/no-explicit-any': 'off', // fix 7 errors
            '@typescript-eslint/no-floating-promises': 'off', // fix 7 errors
            '@typescript-eslint/no-misused-promises': 'off', // more than 100 errors
            '@typescript-eslint/no-redundant-type-constituents': 'off', // fix 3 errors
            '@typescript-eslint/no-unsafe-argument': 'off', // more than 300 errors
            '@typescript-eslint/no-unsafe-return': 'off', // fix 1 error
            // '@typescript-eslint/no-unused-vars': 'off', // fix 4 errors
            '@typescript-eslint/require-await': 'off', // about 20 errors

            '@typescript-eslint/array-type': 'error',
        },
    },
    {
        files: ['*.js', 'tailwind-plugins/*.js'], // non-product files

        languageOptions: {
            ...parserAndOptions,
            globals: {
                ...nodeGlobals,
            },
            sourceType: 'commonjs',
        },

        // Key of plugin is namespace of its rules.
        plugins: {
            import: pluginImport,
        },
        rules: {
            'import/no-extraneous-dependencies': [
                'error',
                {
                    devDependencies: [
                        path.join(__dirname, 'eslint.config.js'),
                        path.join(__dirname, 'postcss.config.js'),
                        path.join(__dirname, 'tailwind.config.js'), // only for @tailwindcss/forms
                    ],
                },
            ],
        },
    },
    {
        files: ['src/**/*.test.{js,ts,tsx}'], // unit tests

        languageOptions: {
            ...parserAndOptions,
            globals: {
                ...jestGlobals,
            },
        },

        // Key of plugin is namespace of its rules.
        plugins: {
            import: pluginImport,
            jest: pluginJest,
            'jest-dom': pluginJestDOM,
            'testing-library': pluginTestingLibrary,
        },
        rules: {
            'import/no-extraneous-dependencies': [
                'error',
                {
                    devDependencies: [path.join(__dirname, 'src/**/*.test.*')],
                },
            ],

            ...pluginJest.configs.recommended.rules,

            'jest/expect-expect': [
                'error',
                {
                    assertFunctionNames: ['expect', 'expectSaga'], // authSagas.test.js integrationSagas.test.js
                },
            ],

            // ...pluginJestDOM.configs.recommended.rules, // TODO fix errors

            ...pluginTestingLibrary.configs.react.rules,

            // TODO remove data-testid attributes from unit tests
            'testing-library/consistent-data-testid': [
                'error',
                {
                    testIdPattern: '^[A-Za-z]+[\\w\\-\\.]*$',
                },
            ],
        },
    },
];
