const path = require('path');

const commonRules = {
    'prettier/prettier': 'error',

    // disallow use of console in favor of raven-js for error capturing
    'no-console': 'error',

    // allow function hoisting
    'no-use-before-define': [
        'error',
        {
            functions: false,
        },
    ],

    curly: [2, 'all'],

    'import/no-extraneous-dependencies': [
        'error',
        {
            devDependencies: [
                path.join(__dirname, '**/*.test.js'),
                path.join(__dirname, 'cypress/**'),
                path.join(__dirname, 'src/setupTests.js'),
                path.join(__dirname, 'src/setupProxy.js'),
                path.join(__dirname, 'tailwind.config.js'),
                path.join(__dirname, 'postcss.config.js'),
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

    // React rules

    'react/display-name': 'warn',
    'react/jsx-props-no-spreading': 'warn',
    'react/static-property-placement': ['warn', 'static public field'],
    'react/prop-types': [
        2,
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
    // forbid arrow functions as well
    'react/jsx-no-bind': [
        'error',
        {
            ignoreRefs: true,
            allowArrowFunctions: false, // Airbnb code style doesn't support ES7 property initializers
            allowBind: false,
        },
    ],
    // stepping away from Airbnb and into more inconsistent world to avoid huge refactoring
    'react/destructuring-assignment': ['off'],
    // forbid 'data-test-id' in preference of 'data-testid'
    'react/forbid-dom-props': [
        'error',
        {
            forbid: ['data-test-id'],
        },
    ],
    'react/no-array-index-key': 'warn', // TODO: eventually switch this rule to error
    'react-hooks/exhaustive-deps': 'warn',

    // DEPRECATED in favor of label-has-associated-control
    // https://github.com/evcohen/eslint-plugin-jsx-a11y/blob/master/docs/rules/label-has-for.md#rule-details
    'jsx-a11y/label-has-for': [0],
    'jsx-a11y/control-has-associated-label': [
        1,
        {
            labelAttributes: ['label'],
            controlComponents: ['Dot', 'Labeled'],
            depth: 3,
        },
    ],

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

const testRules = {
    ...commonRules,

    'jest/no-focused-tests': 'error',
    'jest/expect-expect': [
        'error',
        {
            assertFunctionNames: ['expect', 'expectSaga'],
        },
    ],
    'testing-library/consistent-data-testid': [
        2,
        {
            testIdPattern: '^[A-Za-z]+[\\w\\-\\.]*$',
        },
    ],
};

module.exports = {
    plugins: ['prettier'],
    parser: 'babel-eslint',
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
    rules: commonRules,
    overrides: [
        {
            files: ['src/**/*.js'],
            env: {
                browser: true,
            },
            rules: {
                ...commonRules,

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
            },
        },
        {
            files: ['**/*.ts', '**/*.tsx'],
            plugins: ['@typescript-eslint', 'prettier'],
            parser: '@typescript-eslint/parser',
            extends: [
                'react-app',
                'plugin:react/recommended',
                'plugin:@typescript-eslint/recommended',
                'plugin:@typescript-eslint/recommended-requiring-type-checking',
                'airbnb-typescript',
                'prettier',
                'prettier/@typescript-eslint',
                'prettier/react',
            ],
            env: { browser: true },
            parserOptions: {
                project: './tsconfig.json',
                tsconfigRootDir: __dirname,
            },
            rules: commonRules,
        },
        {
            files: ['src/**/*.test.js'],
            plugins: ['prettier', 'jest'],
            extends: [
                'react-app',
                'airbnb',
                'plugin:testing-library/react',
                'plugin:jest/recommended',
                'plugin:react/recommended',
                'prettier',
                'prettier/react',
            ],
            env: {
                browser: true,
                jest: true,
            },
            rules: testRules,
        },
        {
            files: ['src/**/*.test.ts', 'src/**/*.test.tsx'],
            plugins: ['prettier', 'jest'],
            extends: [
                'react-app',
                'plugin:testing-library/react',
                'plugin:jest/recommended',
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
            rules: testRules,
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

                'jest/no-focused-tests': 'error',
            },
        },
    ],
};
