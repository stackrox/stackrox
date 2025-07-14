/* globals module require */

const path = require('node:path');

// Limited rules have exceptions via ignores property.
// When ESLint plugin for Visual Studio Code has support for suppressions, they might supersede limited rules.

const rules = {
    // ESLint naming convention for positive rules:
    // If your rule is enforcing the inclusion of something, use a short name without a special prefix.

    'export-default-react': {
        // Prevent mistaken assumptions about results from Find in Files.
        meta: {
            type: 'problem',
            docs: {
                description:
                    'Require that file name be consistent with export default of React component name',
            },
            schema: [],
        },
        create(context) {
            return {
                ExportDefaultDeclaration(node) {
                    if (typeof node.declaration?.name === 'string') {
                        const { name } = node.declaration;
                        const { filename } = context;
                        const basenameWithoutExtension = path.basename(
                            filename,
                            path.extname(filename)
                        );
                        if (basenameWithoutExtension !== name) {
                            const ancestors = context.sourceCode.getAncestors(node);
                            const hasReactImportDeclaration = ancestors[0]?.body?.some(
                                (child) =>
                                    child.type === 'ImportDeclaration' &&
                                    child.source?.value === 'react'
                            );
                            // Omit from previous condition, because hooks do not import React.
                            // child.specifiers?.[0]?.local?.name === 'React'
                            if (hasReactImportDeclaration) {
                                context.report({
                                    node,
                                    message: `Require that file name be consistent with export default of React component name: ${name}`,
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
};

// Use limited as key of pluginLimited in eslint.config.js file.

const pluginLimited = {
    meta: {
        name: 'pluginLimited',
        version: '0.0.1',
    },
    rules,
    // config for recommended rules is not relevant
};

module.exports = pluginLimited;
