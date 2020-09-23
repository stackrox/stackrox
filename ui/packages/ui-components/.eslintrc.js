/* eslint @typescript-eslint/no-var-requires: 0 */

const path = require('path');

const commonRules = {
    'prettier/prettier': 'error',

    // disallow use of console in favor of proper error capturing
    'no-console': 'error',

    'react/destructuring-assignment': ['off'],

    curly: ['error', 'all'],

    // forbid 'data-test-id' in preference of 'data-testid'
    'react/forbid-dom-props': [
        'error',
        {
            forbid: ['data-test-id'],
        },
    ],
};

const testRules = {
    ...commonRules,

    'jest/no-focused-tests': 'error',
};

const commonExtensions = [
    'plugin:react/recommended',
    'plugin:@typescript-eslint/recommended',
    'plugin:@typescript-eslint/recommended-requiring-type-checking',
    'plugin:eslint-comments/recommended',
    'airbnb-typescript',
    'prettier',
    'prettier/@typescript-eslint',
    'prettier/react',
];

module.exports = {
    plugins: ['@typescript-eslint', 'prettier', 'jest', 'jest-dom', 'testing-library'],
    parser: '@typescript-eslint/parser',
    extends: commonExtensions,
    parserOptions: {
        project: './tsconfig.eslint.json',
        tsconfigRootDir: __dirname,
    },
    rules: {
        ...commonRules,

        'import/no-extraneous-dependencies': [
            'error',
            {
                devDependencies: [
                    path.join(__dirname, '**/*.test.ts'),
                    path.join(__dirname, '**/*.test.tsx'),
                    path.join(__dirname, '**/*.stories.tsx'),
                    path.join(__dirname, '.storybook/**/*'),
                    path.join(__dirname, '.prettierrc.js'),
                    path.join(__dirname, '.postcssrc.js'),
                    path.join(__dirname, 'tailwind.config.js'),
                    path.join(__dirname, 'jest.config.js'),
                ],
                optionalDependencies: false,
            },
        ],

        'jsx-a11y/label-has-associated-control': [
            2,
            {
                labelAttributes: ['label'],
                controlComponents: ['Field'],
                depth: 3,
            },
        ],
    },

    overrides: [
        {
            files: ['src/**/*'],
            env: {
                browser: true,
            },
        },
        {
            files: ['*.test.ts', '*.test.tsx'],
            extends: [
                ...commonExtensions,
                'plugin:jest/recommended',
                'plugin:jest-dom/recommended',
                'plugin:testing-library/react',
            ],
            env: {
                browser: true,
                jest: true,
            },
            rules: testRules,
        },
        {
            files: ['src/**/*.stories.tsx'],
            rules: {
                ...commonRules,
                // not checking prop types for story components
                'react/prop-types': [
                    'error',
                    {
                        skipUndeclared: true,
                    },
                ],
            },
        },
    ],
};
