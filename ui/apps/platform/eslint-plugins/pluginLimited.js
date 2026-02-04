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

    'feature-flags': {
        // Feature flag must match pattern and occur in alphabetical order
        // to minimize merge conflicts when multiple people add or delete strings.
        // Omit fix function because type union might have comments.
        meta: {
            type: 'problem',
            docs: {
                description: 'Feature flag must match pattern and occur in alphabetical order',
            },
            schema: [],
        },
        create(context) {
            return {
                TSLiteralType(node) {
                    const ancestors = context.sourceCode.getAncestors(node);
                    if (
                        typeof node.literal?.value === 'string' &&
                        ancestors.length >= 2 &&
                        ancestors[ancestors.length - 1].type === 'TSUnionType' &&
                        ancestors[ancestors.length - 2].type === 'TSTypeAliasDeclaration'
                    ) {
                        const ancestor1 = ancestors[ancestors.length - 1];
                        const ancestor2 = ancestors[ancestors.length - 2];
                        if (ancestor2.id?.name === 'FeatureFlagEnvVar') {
                            const { value } = node.literal;
                            if (!value.match(/^ROX(_[A-Z\d]+)+$/)) {
                                context.report({
                                    node,
                                    message:
                                        'Feature flags must match pattern: ROX_ONE_OR_MORE_WORDS',
                                });
                            } else if (
                                Array.isArray(ancestor1.types) &&
                                ancestor1.types.every(
                                    (type) => typeof type?.literal?.value === 'string'
                                )
                            ) {
                                const { types } = ancestor1;
                                const indexOfNode = types.indexOf(node);
                                if (indexOfNode >= 0) {
                                    // Quadratic complexity to identify particular feature flag
                                    // seems acceptable for expected number of feature flags.
                                    const indexFirstNotInAlphabeticalOrder = types.findIndex(
                                        (type, index) =>
                                            index !== 0 &&
                                            types[index - 1].literal.value > type.literal.value
                                    );
                                    if (indexOfNode === indexFirstNotInAlphabeticalOrder) {
                                        context.report({
                                            node,
                                            message:
                                                'Feature flags must occur in alphabetical order',
                                        });
                                    }
                                }
                            }
                        }
                    }
                },
            };
        },
    },
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

    'no-Tailwind': {
        // Forbid Tailwind classes outside legacy folders.
        // See ignores array in eslint.config.js file.
        // See purge array in tailwind.config.js file.
        meta: {
            type: 'problem',
            docs: {
                description: 'Forbid Tailwind classes outside legacy folders',
            },
            schema: [],
        },
        create(context) {
            return {
                JSXOpeningElement(node) {
                    // Add classes from application-specific .css files.
                    const classNamesApplication = [
                        'ConditionTextInput', // Components ConditionText.tsx
                        'acs-m-manual-inclusion', // AccessControl EffectiveAccessScopeTable.tsx
                        'acs-pf-horizontal-subnav', // MainPage HorizontalSubnav.tsx
                        'advanced-filters-toolbar', // Vulnerabilities AdvancedFiltersToolbar.tsx
                        'advanced-flows-filters-select', // NetworkGraph AdvancedFlowsFilter.tsx
                        'certificate-input', // AccessControl ConfigurationFormFields.tsx
                        'cluster-select', // NetworkGraph ClusterSelector.tsx
                        'cluster-status-panel', // Clusters ClusterSummaryGrid.tsx
                        'collection-form-expandable-section', // Collections CollectionForm.tsx
                        'compare-yaml-modal', // NetworkGraph CompareYAMLModal.tsx
                        'cve-severity-select', // Vulnerabilities CVESeverityDropdown.tsx
                        'deployment-select', // NetworkGraph DeploymentSelector.tsx
                        'description', // Policies MitreAttackVectorsFormSection.tsx
                        'draggable-grip', // Policies PolicyCriteriaKey.tsx
                        'dropzone', // Policies PolicySectionDropTarget.tsx
                        'error-boundary-page-column', // Components ErrorBoundaryPage.tsx
                        'error-boundary-stack', // Components ErrorBoundaryPage.tsx
                        'error-boundary-stack-column', // Components ErrorBoundaryPage.tsx
                        'error-boundary-stack-item', // Components ErrorBoundaryPage.tsx
                        'error-boundary-stacks-row', // Components ErrorBoundaryPage.tsx
                        'formatted-text', // CheckDetailsInfo.tsx
                        'json-input', // Integrations Forms folder
                        'loading-section', // Components LoadingSection.tsx
                        'microsoft-sentinel-form', // Integrations MicrosoftSentinelIntegrationForm.tsx
                        'mitre-tactic-item', // Policies MitreAttackVectorsFormSection.tsx
                        'mitre-tactics-list', // Policies MitreAttackVectorsFormSection.tsx
                        'mitre-technique-item', // Policies MitreAttackVectorsFormSection.tsx
                        'mitre-techniques-list', // Policies MitreAttackVectorsFormSection.tsx
                        'namespace-select', // Dashboard NamespaceSelect.tsx
                        'namespace-select', // NetworkGraph NamespaceSelect.tsx

                        // NetworkGraph folder
                        'network-graph',
                        'network-graph-menu-list',
                        'network-graph-selector-bar',

                        'network-policies-generation-scope', // NetworkGraph NetworkPoliciesGenerationScope.tsx
                        'network-policies-yaml', // NetworkGraph NetworkPoliciesYAML.tsx
                        'or-divider', // Policies BooleanPolicyLogicSection.tsx
                        'or-divider-container', // Policies BooleanPolicyLogicSection.tsx
                        'policy-criteria-key', // Policies PolicyCriteriaKey.tsx
                        'policy-enforcement-card', // Policies PolicyEnforcementForm.tsx
                        'policy-section-card-body', // Policies PolicySection.tsx
                        'policy-section-card-header', // Policies PolicySection.tsx
                        'preview-violations', // Policies ReviewPolicyForm.tsx
                        'review-policy', // Policies ReviewPolicyForm.tsx

                        // Collections RuleSelector folder
                        'rule-selector',
                        'rule-selector-add-value-button',
                        'rule-selector-delete-value-button',
                        'rule-selector-label-rule-separator',
                        'rule-selector-list',
                        'rule-selector-list-item',
                        'rule-selector-match-type-select',
                        'rule-selector-name-value-input',

                        'resource-icon', // Components ResourceIcon.tsx
                        'search-filter-labels', // Components CompoundSearchFilterLabels.tsx
                        'severity-count-labels', // Vulnerabilities SeverityCountLabels.tsx
                        'truncate-multiline', // SystemConfig components folder
                        'vm-filter-toolbar-dropdown', // Vulnerabilities components folder
                        'vulnerability-exception-request-overview', // Vulnerabilities RequestOverview.tsx
                        'widget-options-menu', // Dashboard WidgetOptionsMenu.tsx
                        'z-xs-101', // Search SearchPage.tsx
                    ];
                    const isTailwind = (className) =>
                        !className.startsWith('pf-') && !classNamesApplication.includes(className);

                    const attributeClassNameString = node.attributes.find(
                        (attribute) =>
                            attribute.name?.name === 'className' &&
                            typeof attribute.value?.value === 'string' &&
                            attribute.value.value.length !== 0
                    );
                    if (attributeClassNameString) {
                        attributeClassNameString.value.value.split(' ').forEach((className) => {
                            if (isTailwind(className)) {
                                context.report({
                                    node,
                                    message: `className: ${className}`,
                                });
                            }
                        });
                    } else {
                        const attributeClassNameTemplateLiteral = node.attributes.find(
                            (attribute) =>
                                attribute.name?.name === 'className' &&
                                Array.isArray(attribute.value?.expression?.quasis) &&
                                attribute.value.expression.quasis.every(
                                    (quasi) => typeof quasi?.value?.cooked === 'string'
                                )
                        );
                        if (attributeClassNameTemplateLiteral) {
                            attributeClassNameTemplateLiteral.value.expression.quasis.forEach(
                                (quasi) => {
                                    quasi.value.cooked.split(' ').forEach((className) => {
                                        if (className.length !== 0 && isTailwind(className)) {
                                            context.report({
                                                node,
                                                message: `className: ${className}`,
                                            });
                                        }
                                    });
                                }
                            );
                        }
                    }
                },
            };
        },
    },
    'no-feather-icons': {
        // Forbid feather icons outside legacy folders.
        // See ignores array in eslint.config.js file.
        // See purge array in tailwind.config.js file.
        meta: {
            type: 'problem',
            docs: {
                description: 'Forbid feather icons outside legacy folders',
            },
            schema: [],
        },
        create(context) {
            return {
                ImportDeclaration(node) {
                    if (node.source?.value === 'react-feather') {
                        context.report({
                            node,
                            message: 'Replace feather icons with PatternFly icons',
                        });
                    }
                },
            };
        },
    },
    'no-logical-or-preceding-array-or-object': {
        // Consistently write more precise nullish coalescing operator.
        meta: {
            type: 'problem',
            docs: {
                description: 'Replace || with nullish coalescing ??',
            },
            schema: [],
        },
        create(context) {
            return {
                LogicalExpression(node) {
                    if (node.operator === '||') {
                        switch (node.right?.type) {
                            case 'ArrayExpression':
                                context.report({
                                    node,
                                    message: `Replace || with ?? preceding array expression`,
                                });
                                break;
                            case 'ObjectExpression':
                                context.report({
                                    node,
                                    message: `Replace || with ?? preceding object expression`,
                                });
                                break;
                            default:
                                break;
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
