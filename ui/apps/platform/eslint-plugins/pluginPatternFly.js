/* globals module */

const rules = {
    // ESLint naming convention for positive rules:
    // If your rule is enforcing the inclusion of something, use a short name without a special prefix.

    'Label-in-Popover-isClickable': {
        // PatternFly Label in (wrapped by) Popover needs visual indication that it is clickable.
        // PatternFly 5 use style={{ cursor: 'pointer' }} prop.
        // TODO Update if PatternFly 6 isClickable prop has the same effect.
        meta: {
            type: 'problem',
            docs: {
                description: 'Label in Popover needs visual indication that it is clickable',
            },
            schema: [],
        },
        create(context) {
            return {
                JSXElement(node) {
                    if (node.openingElement?.name?.name === 'Label') {
                        const ancestors = context.sourceCode.getAncestors(node);
                        if (
                            ancestors.length >= 1 &&
                            ancestors[ancestors.length - 1].openingElement?.name?.name ===
                                'Popover' &&
                            !node.openingElement?.attributes.some(
                                (attribute) =>
                                    attribute.name?.name === 'style' &&
                                    Array.isArray(attribute.value?.expression?.properties) &&
                                    attribute.value?.expression?.properties.length === 1 &&
                                    attribute.value?.expression?.properties?.[0]?.key?.name ===
                                        'cursor' &&
                                    attribute.value?.expression?.properties?.[0]?.value?.value ===
                                        'pointer'
                            )
                        ) {
                            context.report({
                                node,
                                message:
                                    "Label in Popover needs visual indication that it is clickable via style={{ cursor: 'pointer' }} prop",
                            });
                        }
                    }
                },
            };
        },
    },
    'Td-dataLabel-Th-text': {
        // Require that if Td element has dataLabel prop with string value,
        // then Th element with same index has corresponding text (or screenReaderText).
        //
        // Exceptions:
        // Rule does not apply for column configurations like integration columns.
        // Rule does not apply if Table is not ancestor of Td in rendered elements.
        // Therefore, minimize use of abstrations to render tables.
        meta: {
            type: 'problem',
            docs: {
                description:
                    'Require that if Td element has dataLabel prop with string value, then Th element with same index has corresponding text (or screenReaderText).',
            },
            schema: [],
        },
        create(context) {
            return {
                JSXOpeningElement(node) {
                    // Strict equality here versus endsWith method in isTd predicate below.
                    if (node.name?.name === 'Td') {
                        const dataLabel = node.attributes.find(
                            (attribute) =>
                                attribute.name?.name === 'dataLabel' &&
                                typeof attribute.value?.value === 'string'
                        )?.value?.value;

                        // Another rule reports absence of dataLabel prop in Td element.
                        if (typeof dataLabel === 'string') {
                            const ancestors = context.sourceCode.getAncestors(node);

                            // Replace with findLast method when browser support includes ECMAScript 2023.
                            let ancestorTable;
                            for (let i = ancestors.length - 1; i >= 0; i -= 1) {
                                const ancestor = ancestors[i];
                                if (ancestor.openingElement?.name?.name === 'Table') {
                                    ancestorTable = ancestor;
                                    break;
                                }
                            }

                            if (ancestorTable) {
                                const isThead = (child) =>
                                    child.openingElement?.name?.name === 'Thead';
                                const childThead = ancestorTable.children?.find(isThead);

                                if (childThead) {
                                    const isTr = (child) =>
                                        child.openingElement?.name?.name === 'Tr';
                                    const childTr = childThead.children?.find(isTr);

                                    if (childTr) {
                                        // Element is right of expression even with multiple conditions.
                                        const getElement = (child) =>
                                            child.openingElement ? child : child.expression?.right;

                                        const hasValueAsText = (arg) =>
                                            arg?.children?.some(
                                                (child) =>
                                                    typeof child.value === 'string' &&
                                                    child.value.trim() === dataLabel
                                            );

                                        // PatternFly adds screenReaderText prop of Th element.
                                        // const hasValueAsScreenReaderText = (arg) =>
                                        //     arg?.openingElement?.attributes?.some(
                                        //         (attribute) =>
                                        //             attribute.name?.name === 'screenReaderText' &&
                                        //             attribute.value?.value === dataLabel
                                        //     );
                                        const hasValueAsScreenReaderText = (arg) =>
                                            arg?.children?.some(
                                                (child) =>
                                                    child.openingElement?.name?.name === 'span' &&
                                                    child.openingElement.attributes?.some(
                                                        (attribute) =>
                                                            attribute.name?.name === 'className' &&
                                                            attribute.value?.value ===
                                                                'pf-v5-screen-reader'
                                                    ) &&
                                                    hasValueAsText(child)
                                            );

                                        const hasValue = (arg) =>
                                            hasValueAsText(arg) || hasValueAsScreenReaderText(arg);

                                        const isTh = (arg) =>
                                            typeof arg?.openingElement?.name?.name === 'string' &&
                                            arg.openingElement.name.name.endsWith('Th');
                                        const iTh = childTr.children
                                            ?.filter((child) => isTh(getElement(child)))
                                            .findIndex((child) => hasValue(getElement(child)));

                                        if (iTh >= 0) {
                                            let ancestorTr;
                                            let childOfTr = node;
                                            for (let i = ancestors.length - 1; i >= 0; i -= 1) {
                                                const ancestor = ancestors[i];
                                                if (ancestor.openingElement?.name?.name === 'Tr') {
                                                    ancestorTr = ancestor;
                                                    break;
                                                } else {
                                                    childOfTr = ancestor;
                                                }
                                            }

                                            const isTd = (arg) =>
                                                typeof arg?.openingElement?.name?.name ===
                                                    'string' &&
                                                arg.openingElement.name.name.endsWith('Td');
                                            const iTd = ancestorTr?.children
                                                ?.filter((child) => isTd(getElement(child)))
                                                .indexOf(childOfTr);

                                            if (iTd >= 0 && iTd !== iTh) {
                                                context.report({
                                                    node,
                                                    message: `Td has dataLabel="${dataLabel}" prop and Th element has corresponding text or screen reader text but zero-based index ${iTd} !== ${iTh}`,
                                                });
                                            } else {
                                                // console.log(`dataLabel="${dataLabel}"`); // eslint-disable-line no-console
                                            }
                                        } else {
                                            context.report({
                                                node,
                                                message: `Td has dataLabel="${dataLabel}" prop but no Th element has corresponding text or screen reader text`,
                                            });
                                        }
                                    } else {
                                        // console.log(`dataLabel="${dataLabel}" without Tr ancestor`); // eslint-disable-line no-console
                                    }
                                } else {
                                    // console.log(`dataLabel="${dataLabel}" without Thead ancestor`); // eslint-disable-line no-console
                                }
                            } else {
                                // console.log(`dataLabel="${dataLabel}" without Table ancestor`); // eslint-disable-line no-console
                            }
                        }
                    }
                },
            };
        },
    },
    'version-utility-class': {
        // Require version of PatternFly utility class.
        // PatternFly 4 did not have version number.
        meta: {
            type: 'problem',
            docs: {
                description: 'Require version of PatternFly utility class',
            },
            schema: [],
        },
        create(context) {
            const findErrorMessage = (value) => {
                const pfRegExpArray = [
                    /^pf-u-/, // utility class (at beginning of string)
                    / pf-u-/, // utility class (in middle of string)
                ];
                for (let i = 0; i !== pfRegExpArray.length; i += 1) {
                    const pfRegExp = pfRegExpArray[i];
                    if (pfRegExp.test(value)) {
                        return `PatternFly utility class ${value} lacks version number`;
                    }
                }
                return undefined;
            };

            return {
                Literal(node) {
                    if (typeof node.value === 'string') {
                        const message = findErrorMessage(node.value);
                        if (typeof message === 'string') {
                            context.report({
                                node,
                                message,
                            });
                        }
                    }
                },
                TemplateLiteral(node) {
                    if (Array.isArray(node.quasis)) {
                        node.quasis.forEach((quasi) => {
                            if (typeof quasi.value?.cooked === 'string') {
                                const message = findErrorMessage(quasi.value.cooked);
                                if (typeof message === 'string') {
                                    context.report({
                                        node,
                                        message,
                                    });
                                }
                            }
                        });
                    }
                },
            };
        },
    },
    'version-variable-class': {
        // Require consistent version of PatternFly variable or class.
        meta: {
            type: 'problem',
            docs: {
                description: 'Require consistent version of PatternFly variable or class',
            },
            schema: [],
        },
        create(context) {
            const findErrorMessage = (value) => {
                const versionExpected = '5';
                // Include capturing group for digits in each regular expression.
                const pfRegExpArray = [
                    /^var\(--pf-v(\d+)-/, // variable inside var (at beginning of string)
                    /^--pf-v(\d+)-/, // variable outside var (at beginning of string)
                    /^pf-v(\d+)-/, // class (at beginning of string)
                    / pf-v(\d+)-/, // class (in middle of string)
                ];
                for (let i = 0; i !== pfRegExpArray.length; i += 1) {
                    const pfRegExp = pfRegExpArray[i];
                    const result = pfRegExp.exec(value);
                    if (Array.isArray(result)) {
                        const [, versionReceived] = result;
                        if (versionReceived !== versionExpected) {
                            return `PatternFly variable or class ${value} has inconsistent version ${versionReceived} instead of ${versionExpected}`;
                        }
                    }
                }
                return undefined;
            };

            return {
                Literal(node) {
                    if (typeof node.value === 'string') {
                        const message = findErrorMessage(node.value);
                        if (typeof message === 'string') {
                            context.report({
                                node,
                                message,
                            });
                        }
                    }
                },
                TemplateLiteral(node) {
                    if (Array.isArray(node.quasis)) {
                        node.quasis.forEach((quasi) => {
                            if (typeof quasi.value?.cooked === 'string') {
                                const message = findErrorMessage(quasi.value.cooked);
                                if (typeof message === 'string') {
                                    context.report({
                                        node,
                                        message,
                                    });
                                }
                            }
                        });
                    }
                },
            };
        },
    },

    // ESLint naming convention for negative rules.
    // If your rule only disallows something, prefix it with no.
    // However, we can write forbid instead of disallow as the verb in description and message.

    'no-Label-in-Link': {
        // PatternFly Label in (wrapped by) React Router Link element which is in LabelGroup
        // affects the height other Label element that is wrapped in `Tooltip` element.
        // Text in (wrapped by) Link has side effect to display underline on mouse hover
        // compared to lack of visual indication for Label in Link.
        meta: {
            type: 'problem',
            docs: {
                description: 'Move React Router Link inside PatternFly Label element',
            },
            schema: [],
        },
        create(context) {
            return {
                JSXElement(node) {
                    if (node.openingElement?.name?.name === 'Label') {
                        const ancestors = context.sourceCode.getAncestors(node);
                        if (
                            ancestors.length >= 1 &&
                            ancestors[ancestors.length - 1].openingElement?.name?.name === 'Link'
                        ) {
                            context.report({
                                node,
                                message: 'Move React Router Link inside PatternFly Label element',
                            });
                        }
                    }
                },
            };
        },
    },
    'no-Label-in-Button-in-Popover': {
        // PatternFly Label in (wrapped by) in Button (wrapped by) in Popover which is in LabelGroup
        // affects the height other Label elements.
        meta: {
            type: 'problem',
            docs: {
                description: 'Do not wrap Label in unneeded Button to wrap it in Popover',
            },
            schema: [],
        },
        create(context) {
            return {
                JSXElement(node) {
                    if (node.openingElement?.name?.name === 'Label') {
                        const ancestors = context.sourceCode.getAncestors(node);
                        if (
                            ancestors.length >= 2 &&
                            ancestors[ancestors.length - 1].openingElement?.name?.name ===
                                'Button' &&
                            ancestors[ancestors.length - 2].openingElement?.name?.name === 'Popover'
                        ) {
                            context.report({
                                node,
                                message:
                                    'Do not wrap Label in unneeded Button to wrap it in Popover',
                            });
                        }
                    }
                },
            };
        },
    },
    'no-Td-data-label': {
        // Although Td element renders prop as data-label attribute,
        // require dataLabel prop to simplify other lint rules.
        meta: {
            type: 'problem',
            docs: {
                description: 'Replace data-label with dataLabel prop in Td element',
            },
            schema: [],
        },
        create(context) {
            return {
                JSXOpeningElement(node) {
                    if (typeof node.name?.name === 'string' && node.name.name.endsWith('Td')) {
                        if (
                            node.attributes.some(
                                (attribute) => attribute.name?.name === 'data-label'
                            )
                        ) {
                            context.report({
                                node,
                                message: 'Replace data-label with dataLabel prop in Td element',
                            });
                        }
                    }
                },
            };
        },
    },
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
            return {
                ImportDeclaration(node) {
                    if (
                        typeof node.source?.value === 'string' &&
                        node.source.value.startsWith('@patternfly/react-core') // startsWith also matches deprecated
                    ) {
                        if (
                            node.specifiers.some(
                                (specifier) =>
                                    typeof specifier.imported?.name === 'string' &&
                                    specifier.imported.name.endsWith('Variant')
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
