/* globals module */

const rules = {
    // ESLint naming convention for positive rules:
    // If your rule is enforcing the inclusion of something, use a short name without a special prefix.

    // ESLint naming convention for negative rules.
    // If your rule only disallows something, prefix it with no.
    // However, we can write forbid instead of disallow as the verb in description and message.

    'no-Variant': {
        // Replace Variant enum member with corresponding string literal.
        // Because TypeScript string enumeration is source of truth for prop.
        // For example, replace variant={AlertVariant.info} with variant="info"
        meta: {
            type: 'problem',
            docs: {
                description: 'Replace Variant enum member with corresponding string literal',
            },
            schema: [],
        },
        create(context) {
            // Temporary forbid list limits changed files per contribution.
            const variants = [
                'ClipboardCopyVariant',
                'EmptyStateVariant',
                'ExpandableSectionVariant',
            ];
            return {
                ImportDeclaration(node) {
                    if (node.source.value === '@patternfly/react-core') {
                        if (
                            node.specifiers.some(
                                (specifier) =>
                                    typeof specifier.imported?.name === 'string' &&
                                    // specifier.imported.name.endsWith('Variant')
                                    variants.includes(specifier.imported.name)
                            )
                        ) {
                            context.report({
                                node,
                                message:
                                    'Replace Variant enum member with corresponding string literal',
                            });
                        }
                    }
                },
            };
        },
    },
};

const pluginKey = 'patternfly'; // key of pluginPatternFly in eslint.config.js file

const pluginPatternFly = {
    meta: {
        name: 'pluginPatternFly',
        version: '0.0.1',
    },
    rules,
    // ...pluginPatternFly.configs.recommended.rules means all rules in eslint.config.js file.
    configs: {
        recommended: {
            rules: Object.fromEntries(
                Object.keys(rules).map((ruleKey) => [`${pluginKey}/${ruleKey}`, 'error'])
            ),
        },
    },
};

module.exports = pluginPatternFly;
