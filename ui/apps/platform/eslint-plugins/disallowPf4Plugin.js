const disallowPf4Plugin = {
    meta: {
        name: 'disallow-pf4',
        version: '0.0.1',
    },
    rules: {
        'no-pf4-utility-classes': {
            meta: {
                type: 'problem',
                docs: {
                    description: 'Ensures no accidental use of PatternFly 4 utility classes.',
                },
                fixable: 'code',
                schema: [],
            },
            create(context) {
                return {
                    Literal(node) {
                        if (
                            typeof node.value !== 'string' ||
                            !(node.value.includes(' pf-u-') || node.value.startsWith('pf-u'))
                        ) {
                            return;
                        }

                        context.report({
                            node,
                            message: 'Unexpected PatternFly 4 utility class (use PF5)',
                            fix(fixer) {
                                const fixedValue = node.value.replaceAll(/pf-u-/g, 'pf-v5-u-');
                                return fixer.replaceText(node, `"${fixedValue}"`);
                            },
                        });
                    },
                };
            },
        },
    },
};

module.exports = disallowPf4Plugin;
