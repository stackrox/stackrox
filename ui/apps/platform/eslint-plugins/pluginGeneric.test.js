/**
 * Reference test file for custom ESLint rules.
 *
 * Run with: npm run test:eslint-rules
 */

/* global require, console */
/* eslint-disable no-console */

const { createRuleTester } = require('./test-utils');
const plugin = require('./pluginGeneric');

const ruleTester = createRuleTester();

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

console.log('✓ All tests passed for pluginGeneric rules');
