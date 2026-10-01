/**
 * Tests for accessibility ESLint rules
 *
 * Run with: npm run test:eslint-rules
 */

/* global require, console */
/* eslint-disable no-console */

const { createRuleTester } = require('./test-utils');
const plugin = require('./pluginAccessibility');

const ruleTester = createRuleTester();

// Test the Button-Tooltip-isAriaDisabled rule
ruleTester.run('Button-Tooltip-isAriaDisabled', plugin.rules['Button-Tooltip-isAriaDisabled'], {
    valid: [
        {
            name: 'Button with isAriaDisabled in Tooltip',
            code: `
                <Tooltip content="Help">
                    <Button isAriaDisabled>Click</Button>
                </Tooltip>
            `,
        },
        {
            name: 'Button with isDisabled not in Tooltip',
            code: `<Button isDisabled>Click</Button>`,
        },
        {
            name: 'Button without disabled props in Tooltip',
            code: `
                <Tooltip content="Help">
                    <Button>Click</Button>
                </Tooltip>
            `,
        },
        {
            name: 'Button with isAriaDisabled in ConditionalTooltip',
            code: `
                <ConditionalTooltip content="Help">
                    <Button isAriaDisabled>Click</Button>
                </ConditionalTooltip>
            `,
        },
        {
            name: 'Button with isDisabled={false} in Tooltip',
            code: `
                <Tooltip content="Help">
                    <Button isDisabled={false}>Click</Button>
                </Tooltip>
            `,
        },
        {
            name: 'Nested Button with isAriaDisabled in Tooltip',
            code: `
                <Tooltip content="Help">
                    <div>
                        <Button isAriaDisabled>Click</Button>
                    </div>
                </Tooltip>
            `,
        },
    ],

    invalid: [
        {
            name: 'Button with isDisabled in Tooltip',
            code: `
                <Tooltip content="Help">
                    <Button isDisabled>Click</Button>
                </Tooltip>
            `,
            errors: [
                {
                    message:
                        'Button wrapped in Tooltip should use isAriaDisabled instead of isDisabled to allow keyboard focus for accessibility',
                },
            ],
        },
        {
            name: 'Button with isDisabled in ConditionalTooltip',
            code: `
                <ConditionalTooltip content="Help">
                    <Button isDisabled>Click</Button>
                </ConditionalTooltip>
            `,
            errors: [
                {
                    message:
                        'Button wrapped in Tooltip should use isAriaDisabled instead of isDisabled to allow keyboard focus for accessibility',
                },
            ],
        },
        {
            name: 'Nested Button with isDisabled in Tooltip',
            code: `
                <Tooltip content="Help">
                    <div>
                        <Button isDisabled>Click</Button>
                    </div>
                </Tooltip>
            `,
            errors: [
                {
                    message:
                        'Button wrapped in Tooltip should use isAriaDisabled instead of isDisabled to allow keyboard focus for accessibility',
                },
            ],
        },
        {
            name: 'Button with isDisabled={true} in Tooltip',
            code: `
                <Tooltip content="Help">
                    <Button isDisabled={true}>Click</Button>
                </Tooltip>
            `,
            errors: [
                {
                    message:
                        'Button wrapped in Tooltip should use isAriaDisabled instead of isDisabled to allow keyboard focus for accessibility',
                },
            ],
        },
    ],
});

console.log('✓ All tests passed for pluginAccessibility rules');
