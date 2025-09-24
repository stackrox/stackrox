/* globals __dirname module require */

const fs = require('node:fs');
const path = require('node:path');

// Adapted from getSrcAliases in vite.config.js file.
const srcPath = path.resolve(__dirname, '..', 'src'); // src is sibling of eslint-plugins folder
const srcSubfolders = fs
    .readdirSync(srcPath, { withFileTypes: true })
    .filter((dirent) => {
        // Avoid hidden directories, like `.DS_Store`
        return dirent.isDirectory() && !dirent.name.startsWith('.');
    })
    .map(({ name }) => name);

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
    'no-relative-path-to-src-in-import': {
        // Prerequisite so import statements from other containers can have consistent order.
        // By the way, absence of src in path is magic in project configuration.
        meta: {
            type: 'problem',
            docs: {
                description: 'Replace relative path to subfolder of src with path from subfolder',
            },
            schema: [],
        },
        create(context) {
            return {
                Literal(node) {
                    if (typeof node.value === 'string') {
                        const ancestors = context.sourceCode.getAncestors(node);
                        if (
                            ancestors.length >= 1 &&
                            ancestors[ancestors.length - 1].type === 'ImportDeclaration'
                        ) {
                            // Calculate slashes in filename after base to prevent false positive
                            // for relative path to subfolder like Containers or hooks within a container folder.
                            const baseSuffix = 'ui/apps/platform/';
                            const indexAtStartOfSuffix = context.filename.indexOf(baseSuffix);
                            if (indexAtStartOfSuffix >= 0) {
                                const indexAfterBase = indexAtStartOfSuffix + baseSuffix.length;
                                const filenameAfterBase = context.filename.slice(indexAfterBase);
                                const depth = [...filenameAfterBase.matchAll(/\//g)].length;
                                const relativePrefix = depth === 0 ? './' : '../'.repeat(depth - 1);
                                if (
                                    srcSubfolders.some((srcSubfolder) =>
                                        node.value.startsWith(`${relativePrefix}${srcSubfolder}/`)
                                    )
                                ) {
                                    context.report({
                                        node,
                                        message:
                                            'Replace relative path to subfolder of src with path from subfolder',
                                    });
                                }
                            }
                        }
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
