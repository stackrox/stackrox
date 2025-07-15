/* globals console module require */

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
                    // export default Whatever
                    // export default function Whatever() {}
                    const name = node.declaration?.name ?? node.declaration?.id?.name;
                    if (typeof name === 'string') {
                        const { filename } = context;
                        const extname = path.extname(filename);
                        const basenameWithoutExtension = path.basename(filename, extname);

                        // Use file name extension in case JSX Transform removes import React as cue for component.
                        const isReactComponentOrHook =
                            ['.jsx', '.tsx'].includes(extname) ||
                            basenameWithoutExtension.startsWith('use') ||
                            name.startsWith('use');
                        if (isReactComponentOrHook) {
                            console.log(`${name} ${basenameWithoutExtension} ${extname}`); // eslint-disable-line no-console
                        }
                        if (isReactComponentOrHook && basenameWithoutExtension !== name) {
                            context.report({
                                node,
                                message: `Require that file name be consistent with export default of React component name: ${name}`,
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
