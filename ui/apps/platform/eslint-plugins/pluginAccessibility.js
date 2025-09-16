/* globals module */

const rules = {
    // ESLint naming convention for positive rules:
    // If your rule is enforcing the inclusion of something, use a short name without a special prefix.

    'Alert-component': {
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
                            !node.attributes.some(
                                (nodeAttribute) =>
                                    nodeAttribute.name?.name === 'component' &&
                                    nodeAttribute.value?.value === 'p'
                            )
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
    'Button-Icon-name': {
        // Require prop for aria-label, aria-labelledby, or title attribute to prevent axe DevTools issue:
        // Buttons must have discernable text
        // https://dequeuniversity.com/rules/axe/4.10/button-name
        meta: {
            type: 'problem',
            docs: {
                description:
                    'Require that Button element has aria-label, aria-labelledby, or title prop if its child is an icon',
            },
            schema: [],
        },
        create(context) {
            return {
                JSXElement(node) {
                    if (node.openingElement?.name?.name === 'Button') {
                        // For now at least, does not suppert ternary expression:
                        // {isEditingName ? <CheckIcon /> : <PencilAltIcon />}
                        if (
                            node.children?.some((child) =>
                                child.openingElement?.name?.name?.endsWith('Icon')
                            )
                        ) {
                            if (
                                !node.openingElement?.attributes?.some((attribute) =>
                                    ['aria-label', 'aria-labelledby', 'title'].includes(
                                        attribute.name?.name
                                    )
                                )
                            ) {
                                context.report({
                                    node,
                                    message:
                                        'Require that Button element has aria-label, aria-labelledby, or title prop if its child is an icon',
                                });
                            }
                        }
                    }
                },
            };
        },
    },
    'CardHeader-onExpand-toggleButtonProps': {
        // Require prop for aria-label attribute to prevent axe DevTools issue:
        // Buttons must have discernable text
        // https://dequeuniversity.com/rules/axe/4.10/button-name
        meta: {
            type: 'problem',
            docs: {
                description:
                    'Require that CardHeader element with onExpand has toggleButtonProps prop with aria-label property',
            },
            schema: [],
        },
        create(context) {
            return {
                JSXOpeningElement(node) {
                    if (node.name?.name === 'CardHeader') {
                        if (
                            node.attributes.some(
                                (nodeAttribute) => nodeAttribute.name?.name === 'onExpand'
                            )
                        ) {
                            if (
                                !node.attributes.some(
                                    (nodeAttribute) =>
                                        nodeAttribute.name?.name === 'toggleButtonProps' &&
                                        nodeAttribute.value?.expression?.properties?.some(
                                            (property) => property?.key?.value === 'aria-label'
                                        )
                                )
                            ) {
                                context.report({
                                    node,
                                    message:
                                        'Require that CardHeader element with onExpand has toggleButtonProps prop with aria-label property',
                                });
                            }
                        }
                    }
                },
            };
        },
    },
    'Chart-ariaTitle': {
        // Require prop for aria-labelledby attribute to prevent axe DevTools issue:
        // <svg> elements with an img role must have an alternative text
        // https://dequeuniversity.com/rules/axe/4.10/svg-img-alt
        meta: {
            type: 'problem',
            docs: {
                description: 'Chart element requires ariaTitle prop',
            },
            schema: [],
        },
        create(context) {
            return {
                JSXOpeningElement(node) {
                    if (node.name?.name === 'Chart') {
                        if (
                            !node.attributes.some(
                                (nodeAttribute) => nodeAttribute.name?.name === 'ariaTitle'
                            )
                        ) {
                            context.report({
                                node,
                                message: 'Chart element requires ariaTitle prop',
                            });
                        }
                    }
                },
            };
        },
    },
    'ExpandableSection-isDetached-contentId-toggleId': {
        // Require props to prevent axe DevTools issue:
        // Landmarks should have a unique role or role/label/title (i.e. accessible name) combination
        // https://dequeuniversity.com/rules/axe/4.10/landmark-unique
        meta: {
            type: 'problem',
            docs: {
                description:
                    'ExpandableSection element with isDetached requires contentId and ToggleId props',
            },
            schema: [],
        },
        create(context) {
            return {
                JSXOpeningElement(node) {
                    if (
                        node.name?.name === 'ExpandableSection' &&
                        node.attributes.some(
                            (nodeAttribute) => nodeAttribute.name?.name === 'isDetached'
                        )
                    ) {
                        if (
                            !node.attributes.some(
                                (nodeAttribute) => nodeAttribute.name?.name === 'contentId'
                            ) ||
                            !node.attributes.some(
                                (nodeAttribute) => nodeAttribute.name?.name === 'toggleId'
                            )
                        ) {
                            context.report({
                                node,
                                message:
                                    'ExpandableSection element with isDetached requires contentId and ToggleId props',
                            });
                        }
                    }
                },
            };
        },
    },
    'ExpandableSectionToggle-contentId-toggleId': {
        // Require props to prevent axe DevTools issue:
        // Landmarks should have a unique role or role/label/title (i.e. accessible name) combination
        // https://dequeuniversity.com/rules/axe/4.10/landmark-unique
        meta: {
            type: 'problem',
            docs: {
                description:
                    'ExpandableSectionToggle element requires contentId and toggleId props',
            },
            schema: [],
        },
        create(context) {
            return {
                JSXOpeningElement(node) {
                    if (node.name?.name === 'ExpandableSectionToggle') {
                        if (
                            !node.attributes.some(
                                (nodeAttribute) => nodeAttribute.name?.name === 'contentId'
                            ) ||
                            !node.attributes.some(
                                (nodeAttribute) => nodeAttribute.name?.name === 'toggleId'
                            )
                        ) {
                            context.report({
                                node,
                                message:
                                    'ExpandableSectionToggle element requires contentId and toggleId props',
                            });
                        }
                    }
                },
            };
        },
    },
    'Popover-aria-label': {
        // Require prop to prevent axe DevTools issue:
        // ARIA dialog and alertdialog nodes should have an accessible name
        // https://dequeuniversity.com/rules/axe/4.9/aria-dialog-name
        meta: {
            type: 'problem',
            docs: {
                description: 'Require that Popover element has aria-label prop',
            },
            schema: [],
        },
        create(context) {
            return {
                JSXOpeningElement(node) {
                    if (node.name?.name === 'Popover') {
                        if (
                            !node.attributes.some(
                                (nodeAttribute) => nodeAttribute.name?.name === 'aria-label'
                            )
                        ) {
                            context.report({
                                node,
                                message: 'Require that Popover element has aria-label prop',
                            });
                        }
                    }
                },
            };
        },
    },
    'Tab-empty-contentId': {
        // Require that empty Tab element has tabContentId prop to prevent axe DevTools issue:
        // ARIA attributes must conform to valid values
        // https://dequeuniversity.com/rules/axe/4.10/aria-valid-attr-value
        meta: {
            type: 'problem',
            docs: {
                description: 'Require that empty Tab element has tabContentId prop',
            },
            schema: [],
        },
        create(context) {
            return {
                JSXElement(node) {
                    if (node.openingElement?.name?.name === 'Tab') {
                        if (node.children?.length === 0) {
                            if (
                                !node.openingElement?.attributes?.some(
                                    (nodeAttribute) => nodeAttribute.name?.name === 'tabContentId'
                                )
                            ) {
                                context.report({
                                    node,
                                    message: 'Require that empty Tab element has tabContentId prop',
                                });
                            }
                        }
                    }
                },
            };
        },
    },
    'Th-screenReaderText': {
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
                                !node.openingElement?.attributes?.some((nodeAttribute) =>
                                    ['expand', 'select', 'screenReaderText'].includes(
                                        nodeAttribute.name?.name
                                    )
                                )
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

    'no-Button-Link': {
        // Forbid Button that has Link, HashLink, or a element as child to prevent axe DevTools issue:
        // Interactive controls must not be nested
        // https://dequeuniversity.com/rules/axe/4.10/nested-interactive
        //
        // The rule does not forbid Button element with component={LinkShim} prop for link that looks like a button.
        meta: {
            type: 'problem',
            docs: {
                description: 'Forbid Button that has Link, HashLink, or a element as child',
            },
            schema: [],
        },
        create(context) {
            return {
                JSXElement(node) {
                    if (node.openingElement?.name?.name === 'Button') {
                        if (
                            node.children?.some((child) =>
                                ['Link', 'HashLink', 'a'].includes(child.openingElement?.name?.name)
                            )
                        ) {
                            context.report({
                                node,
                                message:
                                    'Forbid Button that has Link, HashLink, or a element as child',
                            });
                        }
                    }
                },
            };
        },
    },
    'no-Popover-footerContent-headerContent': {
        // Forbid props that cause axe DevTools issues:
        // Heading levels should only increase by one
        // https://dequeuniversity.com/rules/axe/4.9/heading-order
        // Document should not have more than one banner landmark
        // https://dequeuniversity.com/rules/axe/4.9/landmark-no-duplicate-banner
        // Document should not have more than one contentinfo landmark
        // https://dequeuniversity.com/rules/axe/4.9/landmark-no-duplicate-contentinfo
        // Ensures landmarks are unique
        // https://dequeuniversity.com/rules/axe/4.9/landmark-unique
        //
        // Use PopoverBodyContent element to compose footer, or header, or both.
        meta: {
            type: 'problem',
            docs: {
                description: 'Forbid Popover footerContent or headerContent props',
            },
            schema: [],
        },
        create(context) {
            return {
                JSXOpeningElement(node) {
                    if (node.name?.name === 'Popover') {
                        if (
                            node.attributes?.some(
                                (nodeAttribute) =>
                                    nodeAttribute.name?.name === 'footerContent' ||
                                    nodeAttribute.name?.name === 'headerContent'
                            )
                        ) {
                            context.report({
                                node,
                                message:
                                    'Forbid Popover footerContent or headerContent props and use PopoverBodyContent in bodyContent instead',
                            });
                        }
                    }
                },
            };
        },
    },
    'no-Tabs-component': {
        // Forbid Tabs element with component="nav" prop to prevent axe DevTools issue:
        // Landmarks should have a unique role or role/label/title (i.e. accessible name) combination
        // https://dequeuniversity.com/rules/axe/4.10/landmark-unique
        //
        // For the record, accessibility issue is when main element has multiple nav elements.
        // For accessibility, consistency, and semantics, let Tabs render default div element.
        meta: {
            type: 'problem',
            docs: {
                description: 'Forbid Tabs element with component prop',
            },
            schema: [],
        },
        create(context) {
            return {
                JSXOpeningElement(node) {
                    if (node.name?.name === 'Tabs') {
                        if (
                            node.attributes.some(
                                (nodeAttribute) => nodeAttribute.name?.name === 'component'
                            )
                        ) {
                            context.report({
                                node,
                                message: 'Forbid Tabs element with component prop',
                            });
                        }
                    }
                },
            };
        },
    },
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
    'no-Th-aria-label': {
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
                            node.attributes.some(
                                (nodeAttribute) => nodeAttribute.name?.name === 'aria-label'
                            )
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
        name: 'accessibility',
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
