const accessibilityPlugin = {
    meta: {
        name: 'accessibilityPlugin',
        version: '0.0.1',
    },
    rules: {
        'require-Alert-component': {
            // Require alternative markup to prevent axe DevTools issue:
            // Heading levels should only increase by one
            meta: {
                type: 'problem',
                docs: {
                    description: 'Require that Alert element has component="p" prop',
                },
                fixable: 'code',
                schema: [],
            },
            create(context) {
                return {
                    JSXOpeningElement(node) {
                        if (node.name?.name === 'Alert') {
                            if (
                                !node.attributes.some((nodeAttribute) => {
                                    return (
                                        nodeAttribute.name?.name === 'component' &&
                                        nodeAttribute.value?.value === 'p'
                                    );
                                })
                            ) {
                                context.report({
                                    node,
                                    message: 'Alert element requires component="p" prop',
                                });
                            }
                        }
                    },
                };
            },
        },
    },
};

module.exports = accessibilityPlugin;
