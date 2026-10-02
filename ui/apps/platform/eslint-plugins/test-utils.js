/**
 * Shared test utilities for ESLint rule tests
 */

/* global require, module */

const { RuleTester } = require('eslint');
const typescriptParser = require('@typescript-eslint/parser');

/**
 * Creates a RuleTester configured for React/JSX
 */
function createRuleTester() {
    return new RuleTester({
        languageOptions: {
            parserOptions: {
                ecmaVersion: 'latest',
                sourceType: 'module',
                ecmaFeatures: {
                    jsx: true,
                },
            },
        },
    });
}

function createTypeScriptRuleTester() {
    return new RuleTester({
        languageOptions: {
            parser: typescriptParser,
            parserOptions: {
                ecmaVersion: 'latest',
                sourceType: 'module',
                ecmaFeatures: {
                    jsx: true,
                },
            },
        },
    });
}

module.exports = {
    createRuleTester,
    createTypeScriptRuleTester,
};
