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
            'react-app-rewired/**',
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
            // https://github.com/eslint/eslint/blob/main/packages/js/src/configs/eslint-recommended.js
            ...pluginESLint.configs.recommended.rules,

            // Require braces even when block has one statement.
            curly: ['error', 'all'],

            // Forbid use of console in favor of raven-js for error capturing.
            'no-console': 'error',

            // https://github.com/mysticatea/eslint-plugin-eslint-comments/blob/master/lib/configs/recommended.js
            ...pluginESLintComments.configs.recommended.rules,

            // Turn off new rules until after we fix errors in follow-up contributions.
            'eslint-comments/disable-enable-pair': 'off', // fix more than 50 errors

            // https://github.com/import-js/eslint-plugin-import/blob/main/config/errors.js
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

            // Turn on rules from airbnb best-practices config that are not in ESLint recommended.
            // https://github.com/airbnb/javascript/blob/master/packages/eslint-config-airbnb-base/rules/best-practices.js
            'array-callback-return': ['error', { allowImplicit: true }],
            'block-scoped-var': 'error',
            // 'class-methods-use-this' is intentional omission
            'consistent-return': 'error',
            'default-case': ['error', { commentPattern: '^no default$' }],
            'default-case-last': 'error',
            // 'default-param-last' is intentional omission
            'dot-notation': ['error', { allowKeywords: true }],
            eqeqeq: ['error', 'always', { null: 'ignore' }],
            'grouped-accessor-pairs': 'error',
            'guard-for-in': 'error',
            'max-classes-per-file': ['error', 1],
            'no-alert': 'warn',
            'no-caller': 'error',
            'no-constructor-return': 'error',
            'no-else-return': ['error', { allowElseIf: false }], // TODO
            'no-empty-function': [
                'error',
                {
                    allow: ['arrowFunctions', 'functions', 'methods'],
                },
            ],
            'no-eval': 'error',
            'no-extend-native': 'error',
            'no-extra-bind': 'error',
            'no-extra-label': 'error',
            'no-implied-eval': 'error',
            'no-iterator': 'error',
            'no-labels': ['error', { allowLoop: false, allowSwitch: false }],
            'no-lone-blocks': 'error',
            'no-loop-func': 'error',
            'no-multi-str': 'error',
            'no-new': 'error',
            'no-new-func': 'error',
            'no-new-wrappers': 'error',
            'no-octal-escape': 'error',
            'no-param-reassign': [
                'error',
                {
                    props: true,
                    ignorePropertyModificationsFor: [
                        'acc', // for reduce accumulators
                        'accumulator', // for reduce accumulators
                        'e', // for e.returnvalue
                        'ctx', // for Koa routing
                        'context', // for Koa routing
                        'req', // for Express requests
                        'request', // for Express requests
                        'res', // for Express responses
                        'response', // for Express responses
                        '$scope', // for Angular 1 scopes
                        'staticContext', // for ReactRouter context
                    ],
                },
            ],
            'no-proto': 'error',
            'no-restricted-properties': [
                'error',
                {
                    object: 'arguments',
                    property: 'callee',
                    message: 'arguments.callee is deprecated',
                },
                {
                    object: 'global',
                    property: 'isFinite',
                    message: 'Please use Number.isFinite instead',
                },
                {
                    object: 'self',
                    property: 'isFinite',
                    message: 'Please use Number.isFinite instead',
                },
                {
                    object: 'window',
                    property: 'isFinite',
                    message: 'Please use Number.isFinite instead',
                },
                {
                    object: 'global',
                    property: 'isNaN',
                    message: 'Please use Number.isNaN instead',
                },
                {
                    object: 'self',
                    property: 'isNaN',
                    message: 'Please use Number.isNaN instead',
                },
                {
                    object: 'window',
                    property: 'isNaN',
                    message: 'Please use Number.isNaN instead',
                },
                {
                    property: '__defineGetter__',
                    message: 'Please use Object.defineProperty instead.',
                },
                {
                    property: '__defineSetter__',
                    message: 'Please use Object.defineProperty instead.',
                },
                {
                    object: 'Math',
                    property: 'pow',
                    message: 'Use the exponentiation operator (**) instead.',
                },
            ],
            'no-return-assign': ['error', 'always'],
            'no-script-url': 'error',
            'no-self-compare': 'error',
            'no-sequences': 'error',
            'no-throw-literal': 'error',
            'no-unused-expressions': [
                'error',
                {
                    allowShortCircuit: false,
                    allowTernary: false,
                    allowTaggedTemplates: false,
                },
            ],
            'no-useless-concat': 'error',
            'no-useless-return': 'error',
            'no-void': 'error',
            'prefer-promise-reject-errors': ['error', { allowEmptyReject: true }],
            /*
            'prefer-regex-literals': [
                'error',
                {
                    disallowRedundantWrapping: true,
                },
            ],
            */ // decide either is omitted intentionally, fix 3 errors, or disable comments
            // radix: 'error', // TODO comment in after making sure no semantic merge conflict
            'vars-on-top': 'error',
            yoda: 'error',

            // Turn on rules from airbnb errors config that are not in ESLint recommended.
            // https://github.com/airbnb/javascript/blob/master/packages/eslint-config-airbnb-base/rules/errors.js
            'for-direction': 'error',
            'getter-return': ['error', { allowImplicit: true }],
            'no-async-promise-executor': 'error',
            'no-compare-neg-zero': 'error',
            'no-cond-assign': ['error', 'always'],
            'no-constant-condition': 'warn',
            'no-control-regex': 'error',
            'no-debugger': 'error',
            'no-dupe-args': 'error',
            'no-dupe-else-if': 'error',
            'no-dupe-keys': 'error',
            'no-duplicate-case': 'error',
            'no-empty': 'error',
            'no-empty-character-class': 'error',
            'no-ex-assign': 'error',
            'no-extra-boolean-cast': 'error',
            'no-extra-semi': 'error',
            'no-func-assign': 'error',
            'no-import-assign': 'error',
            'no-inner-declarations': 'error',
            'no-invalid-regexp': 'error',
            'no-irregular-whitespace': 'error',
            'no-loss-of-precision': 'error',
            'no-misleading-character-class': 'error',
            'no-obj-calls': 'error',
            'no-prototype-builtins': 'error',
            'no-regex-spaces': 'error',
            'no-setter-return': 'error',
            'no-sparse-arrays': 'error',
            'no-unexpected-multiline': 'error',
            'no-unreachable': 'error',
            'no-unsafe-finally': 'error',
            'no-unsafe-negation': 'error',
            'no-unsafe-optional-chaining': ['error', { disallowArithmeticOperators: true }],
            'no-useless-backreference': 'error',
            'use-isnan': 'error',
            'valid-typeof': ['error', { requireStringLiterals: true }],

            'no-await-in-loop': 'error',
            // 'no-promise-executor-return': 'error', // fix 5 errors
            'no-template-curly-in-string': 'error',
            'no-unreachable-loop': [
                'error',
                {
                    ignore: [],
                },
            ],

            // Turn on rules from airbnb es6 config that are not in ESLint recommended.
            // https://github.com/airbnb/javascript/blob/master/packages/eslint-config-airbnb-base/rules/es6.js
            // 'arrow-body-style' is intentional omission
            'constructor-super': 'error',
            'no-class-assign': 'error',
            'no-const-assign': 'error',
            'no-dupe-class-members': 'error',
            'no-new-symbol': 'error',
            'no-this-before-super': 'error',
            'require-yield': 'error',
            // 'no-restricted-exports' is intentional omission
            'no-useless-computed-key': 'error',
            'no-useless-constructor': 'error',
            'no-useless-rename': [
                'error',
                {
                    ignoreDestructuring: false,
                    ignoreImport: false,
                    ignoreExport: false,
                },
            ],
            'no-var': 'error',
            'object-shorthand': [
                'error',
                'always',
                {
                    ignoreConstructors: false,
                    avoidQuotes: true,
                },
            ],
            'prefer-arrow-callback': [
                'error',
                {
                    allowNamedFunctions: false,
                    allowUnboundThis: true,
                },
            ],
            'prefer-const': [
                'error',
                {
                    destructuring: 'any',
                    ignoreReadBeforeAssign: true,
                },
            ],
            'prefer-destructuring': [
                'error',
                {
                    VariableDeclarator: {
                        array: false,
                        object: true,
                    },
                    AssignmentExpression: {
                        array: true,
                        object: false,
                    },
                },
                {
                    enforceForRenamedProperties: false,
                },
            ],
            'prefer-numeric-literals': 'error',
            'prefer-rest-params': 'error',
            'prefer-spread': 'error',
            'prefer-template': 'error',
            'rest-spread-spacing': ['error', 'never'], // deprecated
            'symbol-description': 'error',

            // Turn on rules from airbnb import config that are not in import errors config.
            // https://github.com/airbnb/javascript/blob/master/packages/eslint-config-airbnb-base/rules/imports.js
            'import/first': 'error',
            'import/newline-after-import': 'error',
            'import/no-absolute-path': 'error',
            'import/no-cycle': ['error', { maxDepth: '∞' }],
            'import/no-duplicates': 'error',
            'import/no-dynamic-require': 'error',
            // 'import/no-extraneous-dependencies' is specified in a more specific configuration
            'import/no-import-module-exports': ['error', { exceptions: [] }],
            'import/no-mutable-exports': 'error',
            // 'import/no-named-as-default' is intentional omission
            // 'import/no-named-as-default-member' is intentional omission
            'import/no-named-default': 'error',
            'import/no-relative-packages': 'error',
            'import/no-self-import': 'error',
            'import/no-useless-path-segments': ['error', { commonjs: true }],
            'import/no-webpack-loader-syntax': 'error',
            'import/order': ['error', { groups: [['builtin', 'external', 'internal']] }],
            // 'import/prefer-default-export' is intentional omission

            // Turn on rules from airbnb style config that are not in ESLint recommended.
            // https://github.com/airbnb/javascript/blob/master/packages/eslint-config-airbnb-base/rules/style.js
            // camelcase is intentional omission
            /*
            // Discuss with team whether or not to turn on this rule
            'max-len': [
                'error',
                100,
                2,
                {
                    ignoreUrls: true,
                    ignoreComments: false,
                    ignoreRegExpLiterals: true,
                    ignoreStrings: true,
                    ignoreTemplateLiterals: true,
                },
            ],
            */
            'new-cap': [
                'error',
                {
                    newIsCap: true,
                    newIsCapExceptions: [],
                    capIsNew: false,
                    capIsNewExceptions: ['Immutable.Map', 'Immutable.Set', 'Immutable.List'],
                },
            ],
            'no-array-constructor': 'error',
            'no-bitwise': 'error',
            'no-continue': 'error',
            'no-multi-assign': ['error'],
            'no-nested-ternary': 'error',
            'no-plusplus': 'error',
            'no-restricted-syntax': [
                'error',
                {
                    selector: 'ForInStatement',
                    message:
                        'for..in loops iterate over the entire prototype chain, which is virtually never what you want. Use Object.{keys,values,entries}, and iterate over the resulting array.',
                },
                {
                    selector: 'ForOfStatement',
                    message:
                        'iterators/generators require regenerator-runtime, which is too heavyweight for this guide to allow them. Separately, loops should be avoided in favor of array iterations.',
                },
                {
                    selector: 'LabeledStatement',
                    message:
                        'Labels are a form of GOTO; using them makes code confusing and hard to maintain and understand.',
                },
                {
                    selector: 'WithStatement',
                    message:
                        '`with` is disallowed in strict mode because it makes code impossible to predict and optimize.',
                },
            ],
            // 'no-underscore-dangle' is intentional omission
            'no-unneeded-ternary': ['error', { defaultAssignment: false }],
            'one-var': ['error', 'never'],
            'operator-assignment': ['error', 'always'],
            'prefer-exponentiation-operator': 'error',
            'prefer-object-spread': 'error',
            'unicode-bom': ['error', 'never'],

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
                // https://github.com/cypress-io/eslint-plugin-cypress/blob/master/index.js
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
                // https://github.com/cypress-io/eslint-plugin-cypress/blob/master/index.js
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

            // Allow chai-style expect(x).to.be.true chain.
            'no-unused-expressions': 'off',

            // https://github.com/cypress-io/eslint-plugin-cypress/blob/master/lib/config/recommended.js
            ...pluginCypress.configs.recommended.rules,

            // Turn off new rules until after we fix errors in follow-up contributions.
            'cypress/no-unnecessary-waiting': 'off', // disable or fix about 8 errors

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
                process: false, // for JavaScript files which have process.env.NODE_ENV and so on
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

            // Turn on rules from airbnb strict config that are not in ESLint recommended.
            // https://github.com/airbnb/javascript/blob/master/packages/eslint-config-airbnb-base/rules/strict.js
            strict: ['error', 'never'], // babel inserts `'use strict';` for us

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

            // https://github.com/jsx-eslint/eslint-plugin-react/blob/master/configs/recommended.js
            ...pluginReact.configs.recommended.rules,

            // Turn off new rules until after we fix errors in follow-up contributions.
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

            // https://github.com/facebook/react/blob/main/packages/eslint-plugin-react-hooks/src/index.js
            ...pluginReactHooks.configs.recommended.rules,

            // 'react-hooks/exhaustive-deps': 'warn', // TODO fix errors and then change from default warn to error?
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
            // https://github.com/typescript-eslint/typescript-eslint/blob/main/packages/eslint-plugin/src/configs/eslint-recommended.ts
            ...pluginTypeScriptESLint.configs['eslint-recommended'].overrides[0].rules,
            // https://github.com/typescript-eslint/typescript-eslint/blob/main/packages/eslint-plugin/src/configs/eslint-recommended.ts
            ...pluginTypeScriptESLint.configs.recommended.rules,
            // https://github.com/typescript-eslint/typescript-eslint/blob/main/packages/eslint-plugin/src/configs/recommended-type-checked.ts
            ...pluginTypeScriptESLint.configs['recommended-type-checked'].rules,

            // Turn off rules from recommended-type-checked configuration because of
            // irrelevant problems when TypeScript modules import from JavaScript modules.
            '@typescript-eslint/no-unsafe-assignment': 'off',
            '@typescript-eslint/no-unsafe-call': 'off',
            '@typescript-eslint/no-unsafe-member-access': 'off',

            // Turn off new rules until after we fix errors in follow-up contributions.
            '@typescript-eslint/no-floating-promises': 'off', // fix 7 errors
            '@typescript-eslint/no-misused-promises': 'off', // more than 100 errors
            '@typescript-eslint/no-unsafe-argument': 'off', // more than 300 errors
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

            ...pluginJestDOM.configs.recommended.rules,

            // https://github.com/testing-library/eslint-plugin-testing-library/blob/main/lib/configs/react.ts
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
