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

// Test the customIcon-ariaHidden rule
ruleTester.run('customIcon-ariaHidden', plugin.rules['customIcon-ariaHidden'], {
    valid: [
        {
            name: 'customIcon element has aria-hidden',
            code: `<Alert customIcon={<Spinner aria-hidden />} title="Waiting" />`,
        },
        {
            name: 'customIcon expression is not JSX',
            code: `<Alert customIcon={statusIcon} title="Waiting" />`,
        },
        {
            name: 'non-customIcon JSX attribute is ignored',
            code: `<Alert icon={<Spinner />} title="Waiting" />`,
        },
    ],

    invalid: [
        {
            name: 'customIcon spinner is missing aria-hidden',
            code: `<Alert customIcon={<Spinner />} title="Waiting" />`,
            errors: [
                {
                    message:
                        'Add aria-hidden to decorative customIcon elements so assistive technologies do not announce duplicated status text',
                },
            ],
        },
        {
            name: 'customIcon icon with other props is missing aria-hidden',
            code: `<Alert customIcon={<CheckCircleIcon color="green" />} title="Complete" />`,
            errors: [
                {
                    message:
                        'Add aria-hidden to decorative customIcon elements so assistive technologies do not announce duplicated status text',
                },
            ],
        },
    ],
});

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
