/**
 * Reference test file for custom ESLint rules.
 *
 * Run with: npm run test:eslint-rules
 */

/* global require, console */
/* eslint-disable no-console */

const { createRuleTester, createTypeScriptRuleTester } = require('./test-utils');
const plugin = require('./pluginGeneric');

const ruleTester = createRuleTester();
const typeScriptRuleTester = createTypeScriptRuleTester();

// Test the Button-LinkShim-href rule
ruleTester.run('Button-LinkShim-href', plugin.rules['Button-LinkShim-href'], {
    valid: [
        {
            name: 'Button with LinkShim and href',
            code: `<Button component={LinkShim} href="/path">Click</Button>`,
        },
        {
            name: 'Button without LinkShim',
            code: `<Button>Click</Button>`,
        },
        {
            name: 'Button with different component',
            code: `<Button component={Link}>Click</Button>`,
        },
    ],

    invalid: [
        {
            name: 'Button with LinkShim but no href',
            code: `<Button component={LinkShim}>Click</Button>`,
            errors: [
                {
                    message:
                        'Require that Button element with component={LinkShim} also has href prop',
                },
            ],
        },
        {
            name: 'Button with LinkShim and other props but no href',
            code: `<Button component={LinkShim} variant="primary">Click</Button>`,
            errors: [
                {
                    message:
                        'Require that Button element with component={LinkShim} also has href prop',
                },
            ],
        },
    ],
});

// Test the JSX-nullish-display-fallback rule
ruleTester.run('JSX-nullish-display-fallback', plugin.rules['JSX-nullish-display-fallback'], {
    valid: [
        {
            name: 'nullish coalescing with literal display fallback',
            code: `<span>{cvssScore ?? '-'}</span>`,
        },
        {
            name: 'boolean fallback expression is not a display fallback',
            code: `<span>{isLoading || hasError}</span>`,
        },
        {
            name: 'element fallback expression is control flow',
            code: `<span>{children || <EmptyState />}</span>`,
        },
    ],

    invalid: [
        {
            name: 'logical OR hides zero when displaying a literal fallback',
            code: `<span>{cvssScore || '-'}</span>`,
            output: `<span>{cvssScore ?? '-'}</span>`,
            errors: [
                {
                    message:
                        'Use ?? for display fallback so valid falsy values like 0 or empty string are not replaced',
                },
            ],
        },
        {
            name: 'logical OR hides empty string when displaying a string fallback',
            code: `<Label>{metadata.name || 'Unknown'}</Label>`,
            output: `<Label>{metadata.name ?? 'Unknown'}</Label>`,
            errors: [
                {
                    message:
                        'Use ?? for display fallback so valid falsy values like 0 or empty string are not replaced',
                },
            ],
        },
        {
            name: 'mixed logical expression is reported but not autofixed',
            code: `<span>{isEnabled && count || '-'}</span>`,
            errors: [
                {
                    message:
                        'Use ?? for display fallback so valid falsy values like 0 or empty string are not replaced',
                },
            ],
        },
    ],
});

// Test the Partial-Record-sparse-map rule
typeScriptRuleTester.run('Partial-Record-sparse-map', plugin.rules['Partial-Record-sparse-map'], {
    valid: [
        {
            name: 'partial record explicitly models missing keys',
            code: `const labels: Partial<Record<string, string>> = { critical: 'Critical' };`,
        },
        {
            name: 'record value can be undefined',
            code: `const labels: Record<string, string | undefined> = { critical: 'Critical' };`,
        },
        {
            name: 'finite key union models total keys',
            code: `const labels: Record<'critical' | 'important', string> = { critical: 'Critical', important: 'Important' };`,
        },
    ],

    invalid: [
        {
            name: 'string record object literal is a sparse map',
            code: `const labels: Record<string, string> = { critical: 'Critical' };`,
            errors: [
                {
                    message:
                        'Use Partial<Record<string, T>> or Record<string, T | undefined> for sparse lookup maps',
                },
            ],
        },
        {
            name: 'string record number map is also sparse',
            code: `const scores: Record<string, number> = {};`,
            errors: [
                {
                    message:
                        'Use Partial<Record<string, T>> or Record<string, T | undefined> for sparse lookup maps',
                },
            ],
        },
    ],
});

console.log('✓ All tests passed for pluginGeneric rules');
