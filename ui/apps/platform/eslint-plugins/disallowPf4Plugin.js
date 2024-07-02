function checkForPf4UtilityClasses(node, value, context) {
    if (typeof value !== 'string' || !value.includes('pf-u-')) {
        return;
    }

    context.report({
        node,
        message: 'Unexpected PatternFly 4 utility class (use PF5 "pf-v5-u-")',
    });
}

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
                        checkForPf4UtilityClasses(node, node.value, context);
                    },
                    TemplateElement(node) {
                        checkForPf4UtilityClasses(node, node.value.raw, context);
                    }
                };
            },
        },
    },
};

module.exports = disallowPf4Plugin;
