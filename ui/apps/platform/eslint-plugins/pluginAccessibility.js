/* globals module */

const rules = {
    // ESLint naming convention for positive rules:
    // If your rule is enforcing the inclusion of something, use a short name without a special prefix.

    'Alert-component-prop': {
        // Require alternative markup to prevent axe DevTools issue:
        // Heading levels should only increase by one
        // https://dequeuniversity.com/rules/axe/4.9/heading-order
        meta: {
            type: 'problem',
            docs: {
                description: 'Require that Alert element has component="p" prop',
            },
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
    'Th-screenReaderText-prop': {
        // Require prop to prevent axe DevTools issue:
        // Table header text should not be empty
        // https://dequeuniversity.com/rules/axe/4.9/empty-table-header

        // Until upgrade to PatternFly 5.3 which has screenReaderText prop,
        // temporary solution is to render child:
        // <span className="pf-v5-screen-reader">{screenReaderText}</span>
        meta: {
            type: 'problem',
            docs: {
                description:
                    'Require that empty Th element has either expand, select, or screenReaderText prop',
            },
            schema: [],
        },
        create(context) {
            return {
                JSXElement(node) {
                    if (node.openingElement?.name?.name === 'Th') {
                        if (node.children?.length === 0) {
                            if (
                                !node.openingElement?.attributes?.some((nodeAttribute) => {
                                    return ['expand', 'select', 'screenReaderText'].includes(
                                        nodeAttribute.name?.name
                                    );
                                })
                            ) {
                                context.report({
                                    node,
                                    message:
                                        'Require that empty Th element has either expand, select, or screenReaderText prop',
                                });
                            }
                        }
                    }
                },
            };
        },
    },

    // ESLint naming convention for negative rules.
    // If your rule only disallows something, prefix it with no.
    // However, we can write forbid instead of disallow as the verb in description and message.

    'no-Td-in-Thead': {
        // Forbid work-around to prevent axe DevTools issue:
        // Table header text should not be empty
        // https://dequeuniversity.com/rules/axe/4.9/empty-table-header
        meta: {
            type: 'problem',
            docs: {
                description: 'Forbid Td as alternative to Th in Thead element',
            },
            schema: [],
        },
        create(context) {
            return {
                JSXElement(node) {
                    if (node.openingElement?.name?.name === 'Td') {
                        const ancestors = context.sourceCode.getAncestors(node);
                        if (
                            ancestors.length >= 2 &&
                            ancestors[ancestors.length - 1].openingElement?.name?.name === 'Tr' &&
                            ancestors[ancestors.length - 2].openingElement?.name?.name === 'Thead'
                        ) {
                            context.report({
                                node,
                                message: 'Forbid Td as alternative to Th in Thead element',
                            });
                        }
                    }
                },
            };
        },
    },
    'no-Th-aria-label-prop': {
        // Forbid work-around to prevent axe DevTools issue:
        // Table header text should not be empty
        // https://dequeuniversity.com/rules/axe/4.9/empty-table-header

        // Until upgrade to PatternFly 5.3 which has screenReaderText prop,
        // temporary solution is to render child:
        // <span className="pf-v5-screen-reader">{screenReaderText}</span>
        meta: {
            type: 'problem',
            docs: {
                description:
                    'Forbid aria-label as alternative to screenReaderText prop in Th element',
            },
            schema: [],
        },
        create(context) {
            return {
                JSXOpeningElement(node) {
                    if (node.name?.name === 'Th') {
                        if (
                            node.attributes.some((nodeAttribute) => {
                                return nodeAttribute.name?.name === 'aria-label';
                            })
                        ) {
                            context.report({
                                node,
                                message:
                                    'Forbid aria-label as alternative to screenReaderText prop in Th element',
                            });
                        }
                    }
                },
            };
        },
    },
};

const pluginKey = 'accessibility'; // key of pluginAccessibility in eslint.config.js file

const pluginAccessibility = {
    meta: {
        name: 'pluginAccessibility',
        version: '0.0.1',
    },
    rules,
    // ...pluginAccessibility.configs.recommended.rules means all rules in eslint.config.js file.
    configs: {
        recommended: {
            rules: Object.fromEntries(
                Object.keys(rules).map((ruleKey) => [`${pluginKey}/${ruleKey}`, 'error'])
            ),
        },
    },
};

module.exports = pluginAccessibility;
