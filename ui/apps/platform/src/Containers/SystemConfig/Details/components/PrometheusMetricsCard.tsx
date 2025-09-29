import React, { ReactElement } from 'react';
import {
    PrometheusMetricsCategory,
    PrivateConfig,
    PrometheusMetricsLabels,
} from 'types/config.proto';
import {
    Card,
    CardBody,
    CardHeader,
    CardTitle,
    DataList,
    DataListCell,
    DataListCheck,
    DataListItem,
    DataListItemCells,
    DataListItemRow,
    DescriptionList,
    DescriptionListDescription,
    DescriptionListGroup,
    DescriptionListTerm,
    Divider,
    EmptyState,
    EmptyStateBody,
    EmptyStateHeader,
    FormGroup,
    FormSection,
    Grid,
    GridItem,
    Label,
    LabelGroup,
    TextInput,
} from '@patternfly/react-core';
import pluralize from 'pluralize';
import { FormikErrors, FormikValues } from 'formik';

const metricPrefixes = {
    imageVulnerabilities: 'rox_central_image_vuln_',
    policyViolations: 'rox_central_policy_violation_',
};

const predefinedMetrics: Record<
    PrometheusMetricsCategory,
    Record<string, PrometheusMetricsLabels>
> = {
    imageVulnerabilities: {
        namespace_severity: { labels: ['Cluster', 'Namespace', 'Severity'] },
        registry_severity: {
            labels: ['Cluster', 'Namespace', 'ImageRegistry', 'Severity'],
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
    },
    policyViolations: {
        namespace_severity: { labels: ['Cluster', 'Namespace', 'Severity'] },
        action: {
            labels: ['Cluster', 'Namespace', 'IsPlatformComponent', 'State', 'Severity', 'Action'],
        },
        stage_severity: {
            labels: [
                'Cluster',
                'Namespace',
                'IsPlatformComponent',
                'Categories',
                'Stage',
                'State',
                'Severity',
            ],
        },
    },
    nodeVulnerabilities: {
        node_severity: {
            labels: ['Cluster', 'Node', 'Severity'],
        },
        component_severity: {
            labels: ['Cluster', 'Node', 'Component', 'IsFixable', 'IsSnoozed', 'Severity'],
        },
    },
};

function labelGroup(labels: PrometheusMetricsLabels): ReactElement {
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

function predefinedMetricListItem(
    enabled: boolean,
    category: PrometheusMetricsCategory,
    metric: string,
    onCustomChange:
        | ((value: unknown, id: string) => Promise<void> | Promise<FormikErrors<FormikValues>>)
        | undefined
): ReactElement {
    return (
        <DataListItem key={`${category}-${metric}`}>
            <DataListItemRow>
                {onCustomChange ? (
                    <DataListCheck
                        key={`${category}-${metric}-checkbox`}
                        id={`${category}-${metric}-checkbox`}
                        aria-labelledby={`${category}-${metric}-label`}
                        name={`${category}-${metric}`}
                        isChecked={enabled}
                        onChange={(_, checked) =>
                            onCustomChange(
                                checked ? predefinedMetrics[category][metric] : undefined,
                                `privateConfig.metrics.${category}.descriptors.${metric}`
                            )
                        }
                    />
                ) : null}
                <DataListItemCells
                    dataListCells={[
                        <DataListCell
                            key={`${category}-${metric}-label`}
                            id={`${category}-${metric}-label`}
                        >
                            <label htmlFor={`${category}-${metric}-checkbox`}>
                                {metricPrefixes[category]}
                                <strong>{metric}</strong>
                            </label>
                        </DataListCell>,
                        <DataListCell key={`${category}-${metric}-predefined`}>
                            Predefined
                        </DataListCell>,
                        <DataListCell key={`${category}-${metric}-descriptors`}>
                            {labelGroup(predefinedMetrics[category][metric])}
                        </DataListCell>,
                    ]}
                />
            </DataListItemRow>
        </DataListItem>
    );
}

// hasMetric checks if the descriptors contain the given metric by looking at
// the metric name and the labels (ignoring the order).
function hasMetric(
    descriptors: Record<string, PrometheusMetricsLabels> | undefined,
    metric: string,
    labels: PrometheusMetricsLabels
): boolean {
    const cfgLabels = descriptors?.[metric]?.labels || [];
    const ll = labels.labels;
    return cfgLabels.length === ll.length && cfgLabels.every((label) => ll.includes(label));
}

function prometheusMetricsDataList(
    descriptors: Record<string, PrometheusMetricsLabels> | undefined,
    category: PrometheusMetricsCategory,
    onCustomChange:
        | ((value: unknown, id: string) => Promise<void> | Promise<FormikErrors<FormikValues>>)
        | undefined
): ReactElement {
    return (
        <DataList aria-label={`${category}-metrics-descriptors`} isCompact>
            {Object.entries(predefinedMetrics[category]).map(
                ([predefinedMetric, originalLabels]) => {
                    // In view mode show only enabled predefined metrics.
                    // In edit mode show all predefined metrics unless they're
                    // overridden.

                    const isEnabledOriginal = hasMetric(
                        descriptors,
                        predefinedMetric,
                        originalLabels
                    );
                    const enabled = descriptors !== undefined && predefinedMetric in descriptors;
                    if (isEnabledOriginal || (onCustomChange && !enabled)) {
                        return predefinedMetricListItem(
                            isEnabledOriginal,
                            category,
                            predefinedMetric,
                            onCustomChange
                        );
                    }
                    return null;
                }
            )}
            {Object.entries(descriptors || {}).map(([metric, labels]) => {
                // Predefined are rendered above.
                if (hasMetric(predefinedMetrics[category], metric, labels)) {
                    return null;
                }
                return (
                    <DataListItem key={`${category}-${metric}`} id={`${category}-${metric}`}>
                        <DataListItemRow>
                            {onCustomChange ? (
                                <DataListCheck
                                    id={metric}
                                    aria-labelledby={metric}
                                    name={metric}
                                    isChecked
                                    isDisabled
                                />
                            ) : null}
                            <DataListItemCells
                                dataListCells={[
                                    <DataListCell>{metric}</DataListCell>,
                                    <DataListCell>Custom</DataListCell>,
                                    <DataListCell>{labelGroup(labels)}</DataListCell>,
                                ]}
                            />
                        </DataListItemRow>
                    </DataListItem>
                );
            })}
        </DataList>
    );
}

export type PrometheusMetricsCardProps = {
    category: PrometheusMetricsCategory;
    period: number;
    descriptors?: Record<string, PrometheusMetricsLabels>;
    title: string;
};

export function PrometheusMetricsCard({
    category,
    period,
    descriptors,
    title,
}: PrometheusMetricsCardProps) {
    const hasMetrics = descriptors && Object.keys(descriptors).length > 0;
    return (
        <GridItem key={category} md={hasMetrics ? 12 : 6} lg={hasMetrics ? 12 : 6}>
            <Card isFlat data-testid={`${category}-view-metrics-config`}>
                <CardHeader
                    actions={{
                        actions: (
                            <>
                                {period && hasMetrics ? (
                                    <Label color="green">Enabled</Label>
                                ) : (
                                    <Label>Disabled</Label>
                                )}
                            </>
                        ),
                        hasNoOffset: false,
                        className: undefined,
                    }}
                >
                    <CardTitle component="h3">{title}</CardTitle>
                </CardHeader>
                <Divider component="div" />
                <CardBody>
                    {hasMetrics ? (
                        <DescriptionList
                            isCompact
                            isHorizontal
                            columnModifier={{
                                default: '1Col',
                            }}
                        >
                            {period ? (
                                <DescriptionListGroup key={`${category}-period`}>
                                    <DescriptionListTerm>Gathering period</DescriptionListTerm>
                                    <DescriptionListDescription>
                                        {period}&nbsp;
                                        {pluralize('minute', period)}
                                    </DescriptionListDescription>
                                </DescriptionListGroup>
                            ) : null}
                            <DescriptionListGroup key={`${category}-metrics`}>
                                <DescriptionListTerm>Metrics</DescriptionListTerm>
                                <DescriptionListDescription>
                                    {prometheusMetricsDataList(descriptors, category, undefined)}
                                </DescriptionListDescription>
                            </DescriptionListGroup>
                        </DescriptionList>
                    ) : (
                        <EmptyState variant="xs">
                            <EmptyStateHeader>No metrics has been configured</EmptyStateHeader>
                            <EmptyStateBody>
                                Edit the configuration, or call <code>/v1/config</code> API to add
                                custom metrics.
                            </EmptyStateBody>
                        </EmptyState>
                    )}
                </CardBody>
            </Card>
        </GridItem>
    );
}

function prometheusMetricsPeriodForm(
    pcfg: PrivateConfig,
    category: PrometheusMetricsCategory,
    onChange: (value, event) => Promise<void> | Promise<FormikErrors<FormikValues>>
): ReactElement {
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
                value={pcfg?.metrics?.[category]?.gatheringPeriodMinutes || 0}
                onChange={(event, value) => onChange(value, event)}
                min={0}
            />
        </FormGroup>
    );
}

export type PrometheusMetricsFormProps = {
    pcfg: PrivateConfig;
    category: PrometheusMetricsCategory;
    title: string;
    onChange: (value, event) => Promise<void> | Promise<FormikErrors<FormikValues>>;
    onCustomChange?: (
        value: unknown,
        id: string
    ) => Promise<void> | Promise<FormikErrors<FormikValues>>;
};

export function PrometheusMetricsForm({
    pcfg,
    category,
    title,
    onChange,
    onCustomChange,
}: PrometheusMetricsFormProps) {
    return (
        <GridItem>
            <Card isFlat data-testid={`${category}-metrics-config`}>
                <CardHeader>
                    <CardTitle component="h3">{title}</CardTitle>
                </CardHeader>
                <Divider component="div" />
                <CardBody>
                    <FormSection>
                        <Grid hasGutter>
                            <GridItem md={12}>
                                {prometheusMetricsPeriodForm(pcfg, category, onChange)}
                            </GridItem>
                            <GridItem md={12}>
                                <FormGroup label="Metrics configuration" role="group">
                                    {prometheusMetricsDataList(
                                        pcfg?.metrics?.[category]?.descriptors,
                                        category,
                                        onCustomChange
                                    )}
                                </FormGroup>
                            </GridItem>
                        </Grid>
                    </FormSection>
                </CardBody>
            </Card>
        </GridItem>
    );
}
