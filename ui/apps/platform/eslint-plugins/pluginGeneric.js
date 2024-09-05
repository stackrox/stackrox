/* globals module */

const rules = {
    // ESLint naming convention for positive rules:
    // If your rule is enforcing the inclusion of something, use a short name without a special prefix.

    'ExternalLink-a': {
        // Require ExternalLink with anchor element as child for consistent presentation of external links.
        meta: {
            type: 'problem',
            docs: {
                description:
                    'Require ExternalLink with anchor element as child for consistent presentation of external links',
            },
            schema: [],
        },
        create(context) {
            return {
                JSXElement(node) {
                    if (node.openingElement?.name?.name === 'a') {
                        const ancestors = context.sourceCode.getAncestors(node);
                        if (
                            ancestors.length >= 1 &&
                            ancestors[ancestors.length - 1].openingElement?.name?.name !==
                                'ExternalLink'
                        ) {
                            context.report({
                                node,
                                message:
                                    'Require ExternalLink with anchor element as child for consistent presentation of external links',
                            });
                        }
                    }
                },
            };
        },
    },
    'Link-target-rel': {
        // Require props for consistent behavior and security of internal links that open in a new tab.
        meta: {
            type: 'problem',
            docs: {
                description:
                    'Require tbat Link element with target="_blank" also has rel="noopener noreferrer" prop',
            },
            schema: [],
        },
        create(context) {
            return {
                JSXOpeningElement(node) {
                    if (node.name?.name === 'Link') {
                        if (
                            node.attributes.some(
                                (nodeAttribute) =>
                                    nodeAttribute.name?.name === 'target' &&
                                    nodeAttribute.value?.value === '_blank'
                            )
                        ) {
                            if (
                                !node.attributes.some(
                                    (nodeAttribute) =>
                                        nodeAttribute.name?.name === 'rel' &&
                                        nodeAttribute.value?.value === 'noopener noreferrer'
                                )
                            ) {
                                context.report({
                                    node,
                                    message:
                                        'Require tbat Link element with target="_blank" also has rel="noopener noreferrer" prop',
                                });
                            }
                        }
                    }
                },
            };
        },
    },
    'a-target-rel': {
        // Require props for consistent behavior and security of external links.
        meta: {
            type: 'problem',
            docs: {
                description:
                    'Require that anchor element has target="_blank" and rel="noopener noreferrer" props',
            },
            schema: [],
        },
        create(context) {
            return {
                JSXOpeningElement(node) {
                    if (node.name?.name === 'a') {
                        if (
                            !node.attributes.some(
                                (nodeAttribute) =>
                                    nodeAttribute.name?.name === 'target' &&
                                    nodeAttribute.value?.value === '_blank'
                            ) ||
                            !node.attributes.some(
                                (nodeAttribute) =>
                                    nodeAttribute.name?.name === 'rel' &&
                                    nodeAttribute.value?.value === 'noopener noreferrer'
                            )
                        ) {
                            context.report({
                                node,
                                message:
                                    'Require that anchor element has target="_blank" and rel="noopener noreferrer" props',
                            });
                        }
                    }
                },
            };
        },
    },

    // ESLint naming convention for negative rules.
    // If your rule only disallows something, prefix it with no.
    // However, we can write forbid instead of disallow as the verb in description and message.
};

const pluginKey = 'generic'; // key of pluginGeneric in eslint.config.js file

const pluginGeneric = {
    meta: {
        name: 'pluginGeneric',
        version: '0.0.1',
    },
    rules,
    // ...pluginGeneric.configs.recommended.rules means all rules in eslint.config.js file.
    configs: {
        recommended: {
            rules: Object.fromEntries(
                Object.keys(rules).map((ruleKey) => [`${pluginKey}/${ruleKey}`, 'error'])
            ),
        },
    },
};

module.exports = pluginGeneric;
