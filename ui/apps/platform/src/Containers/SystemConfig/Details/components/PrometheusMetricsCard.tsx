import { useState } from 'react';
import type { ReactElement } from 'react';
import {
    Badge,
    Card,
    CardBody,
    DescriptionList,
    DescriptionListDescription,
    DescriptionListGroup,
    DescriptionListTerm,
    EmptyState,
    EmptyStateBody,
    FormGroup,
    FormSection,
    Grid,
    GridItem,
    Label,
    LabelGroup,
    Tab,
    TabTitleText,
    Tabs,
    TextInput,
} from '@patternfly/react-core';
import { MinusIcon, PlusIcon } from '@patternfly/react-icons';
import { Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import pluralize from 'pluralize';
import type { FormikErrors, FormikValues } from 'formik';

import type {
    PrivateConfig,
    PrometheusMetricsCategory,
    PrometheusMetricsLabels,
} from 'types/config.proto';

const metricPrefixes = {
    imageVulnerabilities: 'rox_central_image_vuln_',
    nodeVulnerabilities: 'rox_central_node_vuln_',
    policyViolations: 'rox_central_policy_violation_',
    administrativeEvents: 'rox_central_admin_event_',
};

const predefinedMetrics: Record<
    PrometheusMetricsCategory,
    Record<string, PrometheusMetricsLabels>
> = {
    imageVulnerabilities: {
        namespace_severity: {
            labels: ['Cluster', 'Namespace', 'IsPlatformWorkload', 'IsFixable', 'Severity'],
        },
        deployment_severity: {
            labels: [
                'Cluster',
                'Namespace',
                'Deployment',
                'IsPlatformWorkload',
                'IsFixable',
                'Severity',
            ],
        },
        cve_severity: {
            labels: ['Cluster', 'CVE', 'IsPlatformWorkload', 'IsFixable', 'Severity'],
        },
    },
    nodeVulnerabilities: {
        node_severity: {
            labels: ['Cluster', 'Node', 'IsFixable', 'Severity'],
        },
        component_severity: {
            labels: ['Cluster', 'Node', 'Component', 'IsFixable', 'Severity'],
        },
        cve_severity: {
            labels: ['Cluster', 'CVE', 'IsFixable', 'Severity'],
        },
    },
    policyViolations: {
        namespace_severity: {
            labels: ['Cluster', 'Namespace', 'IsPlatformComponent', 'Action', 'Severity'],
            includeFilters: { State: 'ACTIVE' },
        },
        deployment_severity: {
            labels: [
                'Cluster',
                'Namespace',
                'Deployment',
                'IsPlatformComponent',
                'Action',
                'Severity',
            ],
            includeFilters: { State: 'ACTIVE' },
        },
    },
    administrativeEvents: {
        domain_occurrences: {
            labels: ['Type', 'Level', 'Domain'],
        },
        resource_occurrences: {
            labels: ['Type', 'Level', 'Domain', 'ResourceType', 'ResourceName'],
        },
    },
};

export type PrometheusMetricsLabelGroupProps = {
    labels: PrometheusMetricsLabels;
};

function PrometheusMetricsLabelGroup({ labels }: PrometheusMetricsLabelGroupProps): ReactElement {
    return (
        <LabelGroup isCompact numLabels={Infinity}>
            {labels.labels.map((label) => {
                return (
                    <Label isCompact key={label}>
                        {label}
                    </Label>
                );
            })}
        </LabelGroup>
    );
}

export type PrometheusMetricsFilterGroupProps = {
    labels: PrometheusMetricsLabels;
};

function PrometheusMetricsFilterGroup({ labels }: PrometheusMetricsFilterGroupProps): ReactElement {
    const includeEntries = Object.entries(labels.includeFilters ?? {}).sort(([a], [b]) =>
        a.localeCompare(b)
    );
    const excludeEntries = Object.entries(labels.excludeFilters ?? {}).sort(([a], [b]) =>
        a.localeCompare(b)
    );
    return (
        <LabelGroup isCompact numLabels={Infinity}>
            {includeEntries.map(([label, pattern]) => {
                return (
                    <Label isCompact key={`include-${label}`} color="green" icon={<PlusIcon />}>
                        {label}: <code>{pattern}</code>
                    </Label>
                );
            })}
            {excludeEntries.map(([label, pattern]) => {
                return (
                    <Label isCompact key={`exclude-${label}`} color="red" icon={<MinusIcon />}>
                        {label}: <code>{pattern}</code>
                    </Label>
                );
            })}
        </LabelGroup>
    );
}

export type PrometheusMetricsPredefinedMetricTableRowProps = {
    category: PrometheusMetricsCategory;
    enabled: boolean;
    metric: string;
    onCustomChange:
        | ((value: unknown, id: string) => Promise<void> | Promise<FormikErrors<FormikValues>>)
        | undefined;
    rowIndex: number;
    showFilters: boolean;
};

function PrometheusMetricsPredefinedMetricTableRow({
    rowIndex,
    enabled,
    category,
    metric,
    onCustomChange,
    showFilters,
}: PrometheusMetricsPredefinedMetricTableRowProps): ReactElement {
    return (
        <Tr key={`${category}-${metric}-row`}>
            {onCustomChange ? (
                <Td
                    key={`${category}-${metric}-checkbox`}
                    id={`${category}-${metric}-checkbox`}
                    aria-labelledby={`${category}-${metric}-label`}
                    name={`${category}-${metric}`}
                    select={{
                        rowIndex,
                        onSelect: (_event, checked) =>
                            onCustomChange(
                                checked ? predefinedMetrics[category][metric] : undefined,
                                `privateConfig.metrics.${category}.descriptors.${metric}`
                            ),
                        isSelected: enabled,
                        isDisabled: false,
                    }}
                />
            ) : null}
            <Td key={`${category}-${metric}-label`} id={`${category}-${metric}-label`}>
                <label htmlFor={`${category}-${metric}-checkbox`}>
                    {metricPrefixes[category]}
                    <strong>{metric}</strong>
                </label>
            </Td>
            <Td key={`${category}-${metric}-predefined`}>Predefined</Td>
            <Td key={`${category}-${metric}-descriptors`}>
                <PrometheusMetricsLabelGroup labels={predefinedMetrics[category][metric]} />
            </Td>
            {showFilters && (
                <Td key={`${category}-${metric}-filters`}>
                    <PrometheusMetricsFilterGroup labels={predefinedMetrics[category][metric]} />
                </Td>
            )}
        </Tr>
    );
}

// recordsMatch returns true if two maps are equal.
function recordsMatch(a: Record<string, string>, b: Record<string, string>): boolean {
    return (
        Object.keys(a).length === Object.keys(b).length &&
        Object.entries(a).every(([key, val]) => b[key] === val)
    );
}

// elementsMatch returns true if two arrays have same elements ignoring order.
function elementsMatch(a: string[], b: string[]): boolean {
    return a.length === b.length && a.every((value) => b.includes(value));
}

// hasMetric checks if the descriptors contain the given metric by looking at
// the metric name and the labels (ignoring the order).
function hasMetric(
    descriptors: Record<string, PrometheusMetricsLabels> | undefined,
    metric: string,
    labels: PrometheusMetricsLabels
): boolean {
    const base = {
        labels: descriptors?.[metric]?.labels ?? [],
        includeFilters: descriptors?.[metric]?.includeFilters ?? {},
        excludeFilters: descriptors?.[metric]?.excludeFilters ?? {},
    };
    const given = {
        labels: labels.labels,
        includeFilters: labels.includeFilters ?? {},
        excludeFilters: labels.excludeFilters ?? {},
    };
    return (
        elementsMatch(base.labels, given.labels) &&
        recordsMatch(base.includeFilters, given.includeFilters) &&
        recordsMatch(base.excludeFilters, given.excludeFilters)
    );
}

function hasFiltersInLabels(labels: PrometheusMetricsLabels): boolean {
    return (
        Object.keys(labels.includeFilters ?? {}).length > 0 ||
        Object.keys(labels.excludeFilters ?? {}).length > 0
    );
}

function metricsHaveFilters(
    descriptors: Record<string, PrometheusMetricsLabels> | undefined,
    category: PrometheusMetricsCategory,
    editMode: boolean
): boolean {
    // In edit mode, check all predefined metrics.
    // In view mode, only check enabled metrics (those in descriptors).
    const hasFiltersInPredefined = editMode
        ? Object.values(predefinedMetrics[category]).some(hasFiltersInLabels)
        : Object.entries(predefinedMetrics[category]).some(
              ([metric, labels]) =>
                  hasMetric(descriptors, metric, labels) && hasFiltersInLabels(labels)
          );
    const hasFiltersInDescriptors = Object.values(descriptors ?? {}).some(hasFiltersInLabels);
    const showFilters = hasFiltersInPredefined || hasFiltersInDescriptors;
    return showFilters;
}

type PrometheusMetricsTableProps = {
    descriptors: Record<string, PrometheusMetricsLabels> | undefined;
    category: PrometheusMetricsCategory;
    onCustomChange:
        | ((value: unknown, id: string) => Promise<void> | Promise<FormikErrors<FormikValues>>)
        | undefined;
};

function PrometheusMetricsTable({
    descriptors,
    category,
    onCustomChange,
}: PrometheusMetricsTableProps): ReactElement {
    const showFilters = metricsHaveFilters(descriptors, category, onCustomChange !== undefined);

    return (
        <Table aria-label={`${category}-metrics-descriptors`} variant="compact">
            <Thead>
                <Tr key={`${category}-metrics-descriptor-header`}>
                    {onCustomChange ? <Th select={undefined} /> : null}
                    <Th width={30}>Metric name</Th>
                    <Th width={10}>Origin</Th>
                    <Th>Labels</Th>
                    {showFilters && <Th>Filters</Th>}
                </Tr>
            </Thead>
            <Tbody>
                {Object.entries(predefinedMetrics[category]).map(
                    ([predefinedMetric, originalLabels], rowIndex) => {
                        // In view mode show only enabled predefined metrics.
                        // In edit mode show all predefined metrics unless they're
                        // overridden.

                        const isEnabledOriginal = hasMetric(
                            descriptors,
                            predefinedMetric,
                            originalLabels
                        );
                        const enabled =
                            descriptors !== undefined && predefinedMetric in descriptors;
                        if (isEnabledOriginal || (onCustomChange && !enabled)) {
                            return (
                                <PrometheusMetricsPredefinedMetricTableRow
                                    rowIndex={rowIndex}
                                    enabled={isEnabledOriginal}
                                    category={category}
                                    metric={predefinedMetric}
                                    onCustomChange={onCustomChange}
                                    showFilters={showFilters}
                                />
                            );
                        }
                        return null;
                    }
                )}
                {Object.entries(descriptors ?? {}).map(([metric, labels]) => {
                    // Predefined are rendered above.
                    if (hasMetric(predefinedMetrics[category], metric, labels)) {
                        return null;
                    }
                    return (
                        <Tr key={`${category}-${metric}`} id={`${category}-${metric}`}>
                            {onCustomChange ? (
                                <Td
                                    id={metric}
                                    aria-labelledby={metric}
                                    name={metric}
                                    selected
                                    disabled
                                />
                            ) : null}
                            <Td>
                                {metricPrefixes[category]}
                                <strong>{metric}</strong>
                            </Td>
                            <Td>Custom</Td>
                            <Td>
                                <PrometheusMetricsLabelGroup labels={labels} />
                            </Td>
                            {showFilters && (
                                <Td>
                                    <PrometheusMetricsFilterGroup labels={labels} />
                                </Td>
                            )}
                        </Tr>
                    );
                })}
            </Tbody>
        </Table>
    );
}

const categoryTitles: Record<PrometheusMetricsCategory, string> = {
    imageVulnerabilities: 'Image vulnerabilities',
    nodeVulnerabilities: 'Node vulnerabilities',
    policyViolations: 'Policy violations',
    administrativeEvents: 'Administrative events',
};

function getMetricCount(descriptors: Record<string, PrometheusMetricsLabels> | undefined): number {
    return descriptors ? Object.keys(descriptors).length : 0;
}

function MetricCountBadge({ count }: { count: number }): ReactElement | null {
    if (count === 0) {
        return null;
    }
    return (
        <Badge isRead className="pf-v6-u-ml-sm">
            {count}
        </Badge>
    );
}

export type PrometheusMetricsTabbedCardProps = {
    privateConfig: PrivateConfig;
};

export function PrometheusMetricsTabbedCard({
    privateConfig,
}: PrometheusMetricsTabbedCardProps): ReactElement {
    const categories = Object.keys(categoryTitles) as PrometheusMetricsCategory[];
    const [activeTab, setActiveTab] = useState<PrometheusMetricsCategory>(categories[0]);

    return (
        <Card data-testid="prometheus-metrics-config">
            <Tabs
                activeKey={activeTab}
                onSelect={(_event, tabKey) => setActiveTab(tabKey as PrometheusMetricsCategory)}
            >
                {categories.map((category) => {
                    const config = privateConfig?.metrics?.[category];
                    const descriptors = config?.descriptors;
                    const period = config?.gatheringPeriodMinutes || 0;
                    const metricCount = getMetricCount(descriptors);
                    const hasMetrics = metricCount > 0;

                    return (
                        <Tab
                            key={category}
                            eventKey={category}
                            title={
                                <TabTitleText>
                                    {categoryTitles[category]}
                                    <MetricCountBadge count={metricCount} />
                                </TabTitleText>
                            }
                        >
                            <CardBody>
                                {hasMetrics ? (
                                    <>
                                        <DescriptionList
                                            isCompact
                                            isHorizontal
                                            horizontalTermWidthModifier={{
                                                default: '15ch',
                                            }}
                                            columnModifier={{
                                                default: '1Col',
                                            }}
                                        >
                                            {period ? (
                                                <DescriptionListGroup>
                                                    <DescriptionListTerm>
                                                        Gathering period
                                                    </DescriptionListTerm>
                                                    <DescriptionListDescription>
                                                        {period}&nbsp;
                                                        {pluralize('minute', period)}
                                                    </DescriptionListDescription>
                                                </DescriptionListGroup>
                                            ) : null}
                                        </DescriptionList>
                                        <PrometheusMetricsTable
                                            descriptors={descriptors}
                                            category={category}
                                            onCustomChange={undefined}
                                        />
                                    </>
                                ) : (
                                    <EmptyState titleText="No metrics configured" variant="xs">
                                        <EmptyStateBody>
                                            Edit the configuration, or call <code>/v1/config</code>{' '}
                                            API to add custom metrics.
                                        </EmptyStateBody>
                                    </EmptyState>
                                )}
                            </CardBody>
                        </Tab>
                    );
                })}
            </Tabs>
        </Card>
    );
}

type PrometheusMetricsPeriodFormProps = {
    pcfg: PrivateConfig;
    category: PrometheusMetricsCategory;
    onChange: (value, event) => Promise<void> | Promise<FormikErrors<FormikValues>>;
};

function PrometheusMetricsPeriodForm({
    pcfg,
    category,
    onChange,
}: PrometheusMetricsPeriodFormProps): ReactElement {
    return (
        <FormGroup
            label="Gathering period in minutes (set to 0 to disable)"
            isRequired
            fieldId={`privateConfig.metrics.${category}.gatheringPeriodMinutes`}
        >
            <TextInput
                isRequired
                type="number"
                id={`privateConfig.metrics.${category}.gatheringPeriodMinutes`}
                name={`privateConfig.metrics.${category}.gatheringPeriodMinutes`}
                value={pcfg?.metrics?.[category]?.gatheringPeriodMinutes}
                onChange={(event, value) => onChange(value, event)}
                min={0}
            />
        </FormGroup>
    );
}

export type PrometheusMetricsTabbedFormProps = {
    pcfg: PrivateConfig;
    onChange: (value, event) => Promise<void> | Promise<FormikErrors<FormikValues>>;
    onCustomChange?: (
        value: unknown,
        id: string
    ) => Promise<void> | Promise<FormikErrors<FormikValues>>;
};

export function PrometheusMetricsTabbedForm({
    pcfg,
    onChange,
    onCustomChange,
}: PrometheusMetricsTabbedFormProps): ReactElement {
    const categories = Object.keys(categoryTitles) as PrometheusMetricsCategory[];
    const [activeTab, setActiveTab] = useState<PrometheusMetricsCategory>(categories[0]);

    return (
        <Card data-testid="prometheus-metrics-config">
            <Tabs
                activeKey={activeTab}
                onSelect={(_event, tabKey) => setActiveTab(tabKey as PrometheusMetricsCategory)}
            >
                {categories.map((category) => {
                    const metricCount = getMetricCount(pcfg?.metrics?.[category]?.descriptors);

                    return (
                        <Tab
                            key={category}
                            eventKey={category}
                            title={
                                <TabTitleText>
                                    {categoryTitles[category]}
                                    <MetricCountBadge count={metricCount} />
                                </TabTitleText>
                            }
                        >
                            <CardBody>
                                <FormSection>
                                    <Grid hasGutter>
                                        <GridItem md={12}>
                                            <PrometheusMetricsPeriodForm
                                                pcfg={pcfg}
                                                category={category}
                                                onChange={onChange}
                                            />
                                        </GridItem>
                                        <GridItem md={12}>
                                            <FormGroup label="Metrics configuration" role="group">
                                                <PrometheusMetricsTable
                                                    descriptors={
                                                        pcfg?.metrics?.[category]?.descriptors
                                                    }
                                                    category={category}
                                                    onCustomChange={onCustomChange}
                                                />
                                            </FormGroup>
                                        </GridItem>
                                    </Grid>
                                </FormSection>
                            </CardBody>
                        </Tab>
                    );
                })}
            </Tabs>
        </Card>
    );
}
