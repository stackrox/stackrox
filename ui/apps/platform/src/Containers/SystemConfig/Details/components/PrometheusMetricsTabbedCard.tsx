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
    Flex,
    Label,
    LabelGroup,
    Tab,
    TabTitleText,
    Tabs,
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

export const metricPrefixes = {
    imageVulnerabilities: 'rox_central_image_vuln_',
    nodeVulnerabilities: 'rox_central_node_vuln_',
    policyViolations: 'rox_central_policy_violation_',
    administrativeEvents: 'rox_central_admin_event_',
};

export const predefinedMetrics: Record<
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

function PrometheusMetricsLabelGroup({
    labels,
}: {
    labels: PrometheusMetricsLabels;
}): ReactElement {
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

function PrometheusMetricsFilterGroup({
    labels,
}: {
    labels: PrometheusMetricsLabels;
}): ReactElement {
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

function recordsMatch(a: Record<string, string>, b: Record<string, string>): boolean {
    return (
        Object.keys(a).length === Object.keys(b).length &&
        Object.entries(a).every(([key, val]) => b[key] === val)
    );
}

function elementsMatch(a: string[], b: string[]): boolean {
    return a.length === b.length && a.every((value) => b.includes(value));
}

export function hasMetric(
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
    const hasFiltersInPredefined = editMode
        ? Object.values(predefinedMetrics[category]).some(hasFiltersInLabels)
        : Object.entries(predefinedMetrics[category]).some(
              ([metric, labels]) =>
                  hasMetric(descriptors, metric, labels) && hasFiltersInLabels(labels)
          );
    const hasFiltersInDescriptors = Object.values(descriptors ?? {}).some(hasFiltersInLabels);
    return hasFiltersInPredefined || hasFiltersInDescriptors;
}

type PrometheusMetricsPredefinedMetricTableRowProps = {
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

export type PrometheusMetricsTableProps = {
    descriptors: Record<string, PrometheusMetricsLabels> | undefined;
    category: PrometheusMetricsCategory;
    onCustomChange:
        | ((value: unknown, id: string) => Promise<void> | Promise<FormikErrors<FormikValues>>)
        | undefined;
};

export function PrometheusMetricsTable({
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

export const categoryTitles: Record<PrometheusMetricsCategory, string> = {
    imageVulnerabilities: 'Image vulnerabilities',
    nodeVulnerabilities: 'Node vulnerabilities',
    policyViolations: 'Policy violations',
    administrativeEvents: 'Administrative events',
};

export function getMetricCount(
    descriptors: Record<string, PrometheusMetricsLabels> | undefined
): number {
    return descriptors ? Object.keys(descriptors).length : 0;
}

export function MetricCountBadge({ count }: { count: number }): ReactElement {
    return <Badge isRead>{count}</Badge>;
}

export type PrometheusMetricsTabbedCardProps = {
    privateConfig: PrivateConfig;
};

export default function PrometheusMetricsTabbedCard({
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
                                    <Flex gap={{ default: 'gapSm' }}>
                                        {categoryTitles[category]}
                                        <MetricCountBadge count={metricCount} />
                                    </Flex>
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
