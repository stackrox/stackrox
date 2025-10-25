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
    'sort-named-imports': {
        // Sort named imports by imported name (if it differs from local name)
        // which seems more intuitive especially when wrapped.
        // ESLint sort-imports rule sorts by local name (if it differs from imported name).
        //
        // For simplicity, sort case sensitive in Unicode order (upper case precedes lower case).
        meta: {
            type: 'problem',
            docs: {
                description: 'Sort named imports in ascending order',
            },
            fixable: 'code',
            schema: [],
        },
        create(context) {
            return {
                ImportDeclaration(node) {
                    if (
                        Array.isArray(node.specifiers) &&
                        node.specifiers.every(
                            (specifier) => typeof specifier?.imported?.name === 'string'
                        )
                    ) {
                        const { specifiers } = node;
                        if (
                            specifiers.some(
                                (specifier, index) =>
                                    index !== 0 &&
                                    specifiers[index - 1].imported.name > specifier.imported.name
                            )
                        ) {
                            context.report({
                                node,
                                message: 'Sort named imports in ascending order',
                                fix(fixer) {
                                    if (context.sourceCode.getCommentsInside(node).length !== 0) {
                                        return null;
                                    }

                                    // Range array consists of [start, end] similar to arguments of slice method.
                                    const start = specifiers[0].range[0];
                                    const end = specifiers[specifiers.length - 1].range[1];
                                    const range = [start, end];

                                    // Sort by imported name consistent with error criterion above.
                                    const sortCallback = (
                                        { imported: { name: nameA } },
                                        { imported: { name: nameB } }
                                    ) => (nameA < nameB ? -1 : nameA > nameB ? 1 : 0);

                                    // Map sorted specification to its node text and text that follows in original order.
                                    const flatMapCallback = (specifier, index) => [
                                        context.sourceCode.getText(specifier),
                                        // For original order, specifiers[index] instead of specifier!
                                        index === specifiers.length - 1
                                            ? ''
                                            : context.sourceCode
                                                  .getText()
                                                  .slice(
                                                      specifiers[index].range[1],
                                                      specifiers[index + 1].range[0]
                                                  ),
                                    ];

                                    const replacement = [...specifiers]
                                        .sort(sortCallback)
                                        .flatMap(flatMapCallback)
                                        .join('');

                                    return fixer.replaceTextRange(range, replacement);
                                },
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
    'no-default-import-react': {
        // Omit default import of React because it is not needed for JSX tranform.
        meta: {
            type: 'problem',
            docs: {
                description:
                    'Omit default import of React because it is not needed for JSX tranform',
            },
            fixable: 'code',
            schema: [],
        },
        create(context) {
            return {
                ImportDefaultSpecifier(node) {
                    if (node.local?.name === 'React') {
                        const ancestors = context.sourceCode.getAncestors(node);
                        if (
                            ancestors.length >= 1 &&
                            typeof ancestors[ancestors.length - 1].source?.value === 'string' &&
                            // endsWith for edge case: '@testing-library/react' instead of 'react'
                            ancestors[ancestors.length - 1].source.value.endsWith('react')
                        ) {
                            const parent = ancestors[ancestors.length - 1];
                            context.report({
                                node,
                                message:
                                    'Omit default import of React because it is not needed for JSX tranform',
                                fix(fixer) {
                                    const { specifiers } = parent;
                                    if (Array.isArray(specifiers) && specifiers.length !== 0) {
                                        // If default import only, remove import declaration.
                                        if (specifiers.length === 1) {
                                            // Remove node does not remove following newline.
                                            // Command line autofixes secondary errors after primary fixes.
                                            // Integrated development environment requires a second interaction to fix.
                                            return fixer.remove(parent);
                                        }

                                        // Because default import precedes named imports,
                                        // remove from beginning of default import to opening brace.

                                        // Range array consists of [start, end] similar to arguments of slice method.
                                        const startDefaultSpecifier = node.range[0];
                                        const endDefaultSpecifier = node.range[1];
                                        const startNextSpecifier = specifiers[1].range[0];
                                        const end = context.sourceCode
                                            .getText()
                                            .indexOf('{', endDefaultSpecifier);
                                        if (end !== -1 && end < startNextSpecifier) {
                                            return fixer.removeRange([startDefaultSpecifier, end]);
                                        }
                                    }

                                    return null;
                                },
                            });
                        }
                    }
                },
            };
        },
    },
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
                JSXMemberExpression(node) {
                    // For example, React.Fragment
                    if (node.object?.name === 'React' && typeof node.property?.name === 'string') {
                        context.report({
                            node,
                            message: `Replace React qualified name with named import: ${node.property.name}`,
                        });
                    }
                },
                MemberExpression(node) {
                    // For example, React.useState
                    if (node.object?.name === 'React' && typeof node.property?.name === 'string') {
                        context.report({
                            node,
                            message: `Replace React qualified name with named import: ${node.property.name}`,
                        });
                    }
                },
                TSQualifiedName(node) {
                    // For example, React.ReactElement
                    if (node.left?.name === 'React' && typeof node.right?.name === 'string') {
                        context.report({
                            node,
                            message: `Replace React qualified name with named import: ${node.right.name}`,
                        });
                    }
                },
            };
        },
    },
    'no-absolute-path-within-container-in-import': {
        // Prerequisite so import statements within same Containers subfolder can have consistent order.
        // By the way, absence of src in path is magic in project configuration.
        meta: {
            type: 'problem',
            docs: {
                description:
                    'Replace absolute path to same Containers subfolder with relative path',
            },
            schema: [],
        },
        create(context) {
            return {
                Literal(node) {
                    if (typeof node.value === 'string' && !node.value.startsWith('.')) {
                        const ancestors = context.sourceCode.getAncestors(node);
                        if (
                            ancestors.length >= 1 &&
                            ancestors[ancestors.length - 1].type === 'ImportDeclaration'
                        ) {
                            // Calculate slashes in filename after base to prevent false positive
                            // for relative path to subfolder like Containers or hooks within a container folder.
                            const baseSuffix = 'ui/apps/platform/src/Containers/';
                            const indexAtStartOfSuffix = context.filename.indexOf(baseSuffix);
                            if (indexAtStartOfSuffix >= 0) {
                                const indexAfterBase = indexAtStartOfSuffix + baseSuffix.length;
                                const filenameAfterBase = context.filename.slice(indexAfterBase);
                                const indexOfSlash = filenameAfterBase.indexOf('/');
                                if (
                                    indexOfSlash >= 0 &&
                                    node.value.startsWith(
                                        `Containers/${filenameAfterBase.slice(0, indexOfSlash)}/`
                                    )
                                ) {
                                    context.report({
                                        node,
                                        message:
                                            'Replace absolute path to same Containers subfolder with relative path',
                                    });
                                }
                            }
                        }
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
