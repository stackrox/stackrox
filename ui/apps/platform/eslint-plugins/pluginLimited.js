/* globals module require */

const path = require('node:path');

// Limited rules have exceptions via ignores property.
// When ESLint plugin for Visual Studio Code has support for suppressions, they might supersede limited rules.

const rules = {
    // ESLint naming convention for positive rules:
    // If your rule is enforcing the inclusion of something, use a short name without a special prefix.

    'react-export-default': {
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
                    const hookRegExp = /^use[A-Z]/; // Uppercase prevents false match like userWhatever

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
                            hookRegExp.test(basenameWithoutExtension) ||
                            hookRegExp.test(name);
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

    // TODO move rule to pluginGeneric after all errors have been fixed.
    'no-inline-type-imports': {
        // Although @typescript-eslint/consistent-type-imports has options:
        // fixStyle: 'separate-type-imports'
        // fixStyle: 'inline-type-imports'
        // it not (at the moment) have a similar option to enforce only separate type imports.
        meta: {
            type: 'problem',
            docs: {
                description: 'Replace inline type import with separate type import statement',
            },
            schema: [],
        },
        create(context) {
            return {
                ImportSpecifier(node) {
                    if (node.importKind === 'type') {
                        context.report({
                            node,
                            message:
                                'Replace inline type import with separate type import statement',
                        });
                    }
                },
            };
        },
    },
    // TODO move rule to pluginGeneric after all errors have been fixed.
    'no-qualified-name-react': {
        // React.Whatever is possible with default import.
        // For consistency and as prerequisite to replace default import with JSX transform.
        // That is, in addition to namespace import, that has no-import-namespace rule.
        meta: {
            type: 'problem',
            docs: {
                description: 'Replace React qualified name with named import',
            },
            schema: [],
        },
        create(context) {
            return {
                TSQualifiedName(node) {
                    if (node?.left?.name === 'React' && typeof node?.right?.name === 'string') {
                        context.report({
                            node,
                            message: `Replace React qualified name with named import: ${node.right.name}`,
                        });
                    }
                },
            };
        },
    },
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
