/* globals module require */

const path = require('node:path');

const rules = {
    // ESLint naming convention for positive rules:
    // If your rule is enforcing the inclusion of something, use a short name without a special prefix.

    'Button-LinkShim-href': {
        // Enforce assumption about Button and LinkShim elements.
        meta: {
            type: 'problem',
            docs: {
                description:
                    'Require that Button element with component={LinkShim} also has href prop',
            },
            schema: [],
        },
        create(context) {
            return {
                JSXOpeningElement(node) {
                    if (node.name?.name === 'Button') {
                        if (
                            node.attributes.some(
                                (attribute) =>
                                    attribute.name?.name === 'component' &&
                                    attribute.value?.expression?.name === 'LinkShim'
                            )
                        ) {
                            if (
                                !node.attributes.some(
                                    (attribute) => attribute.name?.name === 'href'
                                )
                            ) {
                                context.report({
                                    node,
                                    message:
                                        'Require that Button element with component={LinkShim} also has href prop',
                                });
                            }
                        }
                    }
                },
            };
        },
    },
    'ExternalLink-anchor': {
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
                    'Require that Link element with target="_blank" also has rel="noopener noreferrer" prop',
            },
            schema: [],
        },
        create(context) {
            return {
                JSXOpeningElement(node) {
                    if (node.name?.name === 'Link') {
                        if (
                            node.attributes.some(
                                (attribute) =>
                                    attribute.name?.name === 'target' &&
                                    attribute.value?.value === '_blank'
                            )
                        ) {
                            if (
                                !node.attributes.some(
                                    (attribute) =>
                                        attribute.name?.name === 'rel' &&
                                        attribute.value?.value === 'noopener noreferrer'
                                )
                            ) {
                                context.report({
                                    node,
                                    message:
                                        'Require that Link element with target="_blank" also has rel="noopener noreferrer" prop',
                                });
                            }
                        }
                    }
                },
            };
        },
    },
    'Td-defaultColumns': {
        // Require that Td element has key and title from defaultColumns configuration.
        // That is, if Td element has props for column management:
        // className={getVisibilityClass('whichever')}
        // dataLabel="Whatever"
        // Then defaultColumns has whichever: {title: "Whatever"} property.
        meta: {
            type: 'problem',
            docs: {
                description:
                    'Require that Td element has key and title from defaultColumns configuration',
            },
            schema: [],
        },
        create(context) {
            return {
                JSXOpeningElement(node) {
                    if (node.name?.name === 'Td') {
                        const classNameJSXAttribute = node.attributes.find(
                            (attribute) =>
                                attribute.name?.name === 'className' &&
                                attribute.value?.expression?.callee?.name ===
                                    'getVisibilityClass' &&
                                attribute.value?.expression?.arguments?.length === 1 &&
                                typeof attribute.value.expression.arguments[0]?.value === 'string'
                        );
                        if (classNameJSXAttribute) {
                            const ancestors = context.sourceCode.getAncestors(node);
                            const defaultColumnsVariableDeclaration = ancestors[0]?.body?.find(
                                (item) =>
                                    item?.declaration?.declarations?.[0]?.id?.name ===
                                    'defaultColumns'
                            );
                            if (!defaultColumnsVariableDeclaration) {
                                context.report({
                                    node,
                                    message: `Td has className={getVisibilityClass(…)} but file does not have defaultColumns`,
                                });
                            } else {
                                const argument =
                                    classNameJSXAttribute.value.expression.arguments[0].value;
                                const defaultColumnProperty =
                                    defaultColumnsVariableDeclaration.declaration.declarations[0].init?.expression?.properties?.find(
                                        (property) => property.key?.name === argument
                                    );
                                if (!defaultColumnProperty) {
                                    context.report({
                                        node,
                                        message: `Td has className={getVisibilityClass("${argument}")} but argument is not a key in defaultColumns`,
                                    });
                                } else {
                                    const dataLabelAttribute = node.attributes.find(
                                        (attribute) =>
                                            attribute.name?.name === 'dataLabel' &&
                                            typeof attribute.value?.value === 'string'
                                    );
                                    // Another rule reports absence of dataLabel prop in Td element.
                                    if (dataLabelAttribute) {
                                        const dataLabel = dataLabelAttribute.value.value;
                                        const title = defaultColumnProperty.value?.properties?.find(
                                            (property) => property.key?.name === 'title'
                                        )?.value?.value;
                                        // TypeScript reports absence of title property in default column property.
                                        if (dataLabel !== title) {
                                            // Another rule reports inconsistency between Td dataLabel and Th text.
                                            context.report({
                                                node,
                                                message: `Td has dataLabel="${dataLabel}" but defaultColumns has ${argument}: {title: "${title}"}`,
                                            });
                                        }
                                    }
                                }
                            }
                        }
                    }
                },
            };
        },
    },
    'Th-defaultColumns': {
        // Require that Th element has key and text from defaultColumns configuration.
        // That is, if Th element has prop for column management:
        // className={getVisibilityClass('whichever')}
        // Whatever as text
        // Then defaultColumns has whichever: {title: "Whatever"} property.
        meta: {
            type: 'problem',
            docs: {
                description:
                    'Require that Th element has key and text from defaultColumns configuration',
            },
            schema: [],
        },
        create(context) {
            return {
                JSXOpeningElement(node) {
                    if (typeof node.name?.name === 'string' && node.name.name.endsWith('Th')) {
                        const classNameJSXAttribute = node.attributes.find(
                            (attribute) =>
                                attribute.name?.name === 'className' &&
                                attribute.value?.expression?.callee?.name ===
                                    'getVisibilityClass' &&
                                attribute.value?.expression?.arguments?.length === 1 &&
                                typeof attribute.value.expression.arguments[0]?.value === 'string'
                        );
                        if (classNameJSXAttribute) {
                            const ancestors = context.sourceCode.getAncestors(node);
                            const defaultColumnsVariableDeclaration = ancestors[0]?.body?.find(
                                (item) =>
                                    item?.declaration?.declarations?.[0]?.id?.name ===
                                    'defaultColumns'
                            );
                            if (!defaultColumnsVariableDeclaration) {
                                context.report({
                                    node,
                                    message: `Th has className={getVisibilityClass(…)} but file does not have defaultColumns`,
                                });
                            } else {
                                const argument =
                                    classNameJSXAttribute.value.expression.arguments[0].value;
                                const defaultColumnProperty =
                                    defaultColumnsVariableDeclaration.declaration.declarations[0].init?.expression?.properties?.find(
                                        (property) => property.key?.name === argument
                                    );
                                if (!defaultColumnProperty) {
                                    context.report({
                                        node,
                                        message: `Th has className={getVisibilityClass("${argument}")} but argument is not a key in defaultColumns`,
                                    });
                                } else {
                                    const parent = ancestors[ancestors.length - 1];
                                    const value = parent.children?.find(
                                        (child) => typeof child.value === 'string'
                                    )?.value;
                                    if (typeof value === 'string') {
                                        const text = value.trim();
                                        const title = defaultColumnProperty.value?.properties?.find(
                                            (property) => property.key?.name === 'title'
                                        )?.value?.value;
                                        // TypeScript reports absence of title property in default column property.
                                        if (text !== title) {
                                            // Another rule reports inconsistency between Td dataLabel and Th text.
                                            context.report({
                                                node,
                                                message: `Th has "${text}" as text but defaultColumns has ${argument}: {title: "${title}"}`,
                                            });
                                        }
                                    }
                                }
                            }
                        }
                    }
                },
            };
        },
    },
    'anchor-target-rel': {
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
                                (attribute) =>
                                    attribute.name?.name === 'target' &&
                                    attribute.value?.value === '_blank'
                            ) ||
                            !node.attributes.some(
                                (attribute) =>
                                    attribute.name?.name === 'rel' &&
                                    attribute.value?.value === 'noopener noreferrer'
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
    'getVersionedDocs-subPath': {
        meta: {
            type: 'problem',
            docs: {
                description:
                    'Require that optional subPath argument of getVersionedDocs function call has no obvious problems',
            },
            schema: [],
        },
        create(context) {
            return {
                CallExpression(node) {
                    if (
                        node.callee?.name === 'getVersionedDocs' &&
                        Array.isArray(node.arguments) &&
                        typeof node.arguments[1]?.value === 'string'
                    ) {
                        const { value } = node.arguments[1];
                        if (value.startsWith('/')) {
                            context.report({
                                node,
                                message:
                                    'Omit leading slash from relative subPath argument of getVersionedDocs function call',
                            });
                        }
                        if (value.includes('.html')) {
                            context.report({
                                node,
                                message:
                                    'Omit .html extension from relative subPath argument of getVersionedDocs function call',
                            });
                        }
                    }
                },
            };
        },
    },
    'import-type-order': {
        // Require import type to follow corresponding import statement (if it exists).
        meta: {
            type: 'problem',
            docs: {
                description:
                    'Require import type to follow corresponding import statement (if it exists).',
            },
            schema: [],
        },
        create(context) {
            return {
                ImportDeclaration(node) {
                    if (node.importKind === 'type' && typeof node.source?.value === 'string') {
                        const ancestors = context.sourceCode.getAncestors(node);
                        if (
                            ancestors.length >= 1 &&
                            Array.isArray(ancestors[ancestors.length - 1].body)
                        ) {
                            const hasImportKindTypeAndSourceValue = (child) =>
                                child.type === 'ImportDeclaration' &&
                                child.importKind === 'type' &&
                                child.source?.value === node.source.value;
                            const hasImportKindValueAndSourceValue = (child) =>
                                child.type === 'ImportDeclaration' &&
                                child.importKind === 'value' &&
                                child.source?.value === node.source.value;

                            const { body } = ancestors[ancestors.length - 1];
                            const indexType = body.findIndex(hasImportKindTypeAndSourceValue);
                            const indexValue = body.findIndex(hasImportKindValueAndSourceValue);
                            if (indexType >= 0 && indexValue >= 0 && indexType !== indexValue + 1) {
                                context.report({
                                    node,
                                    message:
                                        'Move import type to follow corresponding import statement',
                                });
                            }
                        }
                    }
                },
            };
        },
    },
    'pagination-function-call': {
        // Require that pagination property has function call like getPaginationParams.
        // Some classic pages have queryService.getPagination function call instead.
        meta: {
            type: 'problem',
            docs: {
                description:
                    'Require that pagination property has function call like getPaginationParams',
            },
            schema: [],
        },
        create(context) {
            return {
                Property(node) {
                    if (node.key?.name === 'pagination' && !node.shorthand) {
                        if (node.value?.type !== 'CallExpression') {
                            context.report({
                                node,
                                message:
                                    'Require that pagination property has function call like getPaginationParams',
                            });
                        }
                    }
                },
            };
        },
    },
    'react-props-name': {
        // Prevent mistaken assumptions about results from Find in Files.
        // For the most common component definition patterns in TypeScript React files:
        //
        // function Whatever(({ … }: WhateverProps) {}
        // export default Whatever;
        //
        // export default function Whatever({ … }: WhateverProps) {}
        //
        // function Whatever({ … }: WhateverProps) {}
        //
        // Use eslint-disable comment if application-specific component has PatternFly props name.
        meta: {
            type: 'problem',
            docs: {
                description: 'Require that React component has consistent props name',
            },
            schema: [],
        },
        create(context) {
            return {
                TSTypeReference(node) {
                    const nameOfPropsType = node?.typeName?.name;
                    const upperRegExp = /^[A-Z]/; // usual convention for React component name
                    if (
                        typeof nameOfPropsType === 'string' &&
                        path.extname(context.filename) === '.tsx'
                    ) {
                        const ancestors = context.sourceCode.getAncestors(node);
                        if (
                            Array.isArray(ancestors) &&
                            ancestors.length > 3 &&
                            ancestors[ancestors.length - 1].type === 'TSTypeAnnotation' &&
                            ancestors[ancestors.length - 2].type === 'ObjectPattern' &&
                            ancestors[ancestors.length - 3].type === 'FunctionDeclaration'
                        ) {
                            const name = ancestors[ancestors.length - 3]?.id?.name;
                            if (typeof name === 'string' && upperRegExp.test(name)) {
                                // specialized case: WhateverIntegrationFormProps endsWith IntegrationFormProps
                                // current baseline: Whatever endsWith WhateverProps
                                // possible minimal: WhateverProps endsWith Props
                                if (!`${name}Props`.endsWith(nameOfPropsType)) {
                                    context.report({
                                        node,
                                        message: `Require that React component ${name} has consistent props name: ${nameOfPropsType}`,
                                    });
                                }
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

    'no-anchor-href-docs-string': {
        // Full path string lacks what getVersionedDocs function provides:
        // Include version number so doc page corresponds to product version.
        // Encapsulate openshift versus redhat in origin of path.
        meta: {
            type: 'problem',
            docs: {
                description:
                    'Replace string with getVersionedDocs function call in href prop of anchor element for product docs',
            },
            schema: [],
        },
        create(context) {
            const isDocs = (href) =>
                href.startsWith('https://docs.openshift.com/acs/') ||
                href.startsWith(
                    'https://docs.redhat.com/en/documentation/red_hat_advanced_cluster_security_for_kubernetes/'
                );
            return {
                JSXOpeningElement(node) {
                    if (node.name?.name === 'a') {
                        if (
                            node.attributes.some(
                                (attribute) =>
                                    attribute.name?.name === 'href' &&
                                    typeof attribute.value?.value === 'string' &&
                                    isDocs(attribute.value.value)
                            )
                        ) {
                            context.report({
                                node,
                                message:
                                    'Replace full path string with getVersionedDocs function call in href prop of anchor element for product docs',
                            });
                        }
                    }
                },
            };
        },
    },
    'no-import-namespace': {
        // Replace namespace with named import (except for yup package).
        // In addition to consistency, minimize false negatives,
        // because import/namespace is slow rule turned off for lint:fast-dev command.
        meta: {
            type: 'problem',
            docs: {
                description: 'Replace namespace with named import (except for yup package)',
            },
            schema: [],
        },
        create(context) {
            return {
                ImportNamespaceSpecifier(node) {
                    const ancestors = context.sourceCode.getAncestors(node);
                    if (ancestors.length >= 1) {
                        const parent = ancestors[ancestors.length - 1];
                        if (typeof parent?.source?.value === 'string') {
                            if (parent.source.value !== 'yup') {
                                context.report({
                                    node,
                                    message:
                                        'Replace namespace with named import (except for yup package)',
                                });
                            } else if (
                                typeof node.local?.name === 'string' &&
                                node.local.name !== parent.source.value
                            ) {
                                context.report({
                                    node,
                                    message: `Use namespace that is consistent with package name: ${parent.source.value}`,
                                });
                            }
                        }
                    }
                },
            };
        },
    },
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
