import React, { ReactElement } from 'react';
import { Category, PrivateConfig, Labels } from 'types/config.proto';
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

const predefinedMetrics: Record<Category, Record<string, Labels>> = {
    imageVulnerabilities: {
        image_vuln_namespace_severity: { labels: ['Cluster', 'Namespace', 'Severity'] },
        image_vuln_deployment_severity: {
            labels: ['Cluster', 'Namespace', 'Deployment', 'Severity'],
        },
        image_vuln_user_workload_severity: {
            labels: ['Cluster', 'Namespace', 'Deployment', 'IsPlatformWorkload', 'Severity'],
        },
    },
};

function labelGroup(labels: Labels): ReactElement {
    return (
        <LabelGroup isCompact numLabels={Infinity}>
            {Object.values(labels.labels).map((label) => {
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
    category: Category,
    metric: string,
    onCustomChange:
        | ((value: unknown, id: string) => Promise<void> | Promise<FormikErrors<FormikValues>>)
        | undefined
): ReactElement {
    return (
        <DataListItem>
            <DataListItemRow>
                <DataListItemCells
                    dataListCells={[
                        onCustomChange ? (
                            <DataListCheck
                                id={`${category}-${metric}-checkbox`}
                                aria-labelledby={`${category}-${metric}`}
                                name={`${category}-${metric}`}
                                isChecked={enabled}
                                onChange={(_, checked) =>
                                    onCustomChange(
                                        checked ? predefinedMetrics[category][metric] : null,
                                        `privateConfig.metrics.${category}.metrics.${metric}`
                                    )
                                }
                            />
                        ) : (
                            <></>
                        ),
                        <DataListCell>{metric}</DataListCell>,
                        <DataListCell>Predefined</DataListCell>,
                        <DataListCell>
                            {labelGroup(predefinedMetrics[category][metric])}
                        </DataListCell>,
                    ]}
                />
            </DataListItemRow>
        </DataListItem>
    );
}

// isIn checks if the metric is one of the predefined ones by looking at
// the metric name, labels (ignoring the order).
// NB: Returns false if the metric is not found in the actual configuration.
function hasMetric(
    cfg: Record<string, Labels> | undefined,
    metric: string,
    labels: string[]
): boolean {
    const cfgLabels = cfg?.[metric]?.labels || [];
    return cfgLabels.length === labels.length && cfgLabels.every((label) => labels.includes(label));
}

function prometheusMetricsDataList(
    mcfg: Record<string, Labels> | undefined,
    category: Category,
    onCustomChange:
        | ((value: unknown, id: string) => Promise<void> | Promise<FormikErrors<FormikValues>>)
        | undefined
): ReactElement {
    return (
        <DataList aria-label={`${category}-metrics-configuration`} isCompact>
            {Object.entries(predefinedMetrics[category]).map(
                ([predefinedMetric, originalLabels]) => {
                    // In view mode show only enabled predefined metrics.
                    // In edit mode show all predefined metrics unless they're
                    // overridden.

                    const isEnabledOriginal = hasMetric(
                        mcfg,
                        predefinedMetric,
                        originalLabels.labels
                    );
                    const enabled = mcfg !== undefined && predefinedMetric in mcfg;
                    if (isEnabledOriginal || (onCustomChange && !enabled)) {
                        return predefinedMetricListItem(
                            isEnabledOriginal,
                            category,
                            predefinedMetric,
                            onCustomChange
                        );
                    }
                    return <></>;
                }
            )}
            {Object.entries(mcfg || {}).map(([metric, labels]) => {
                // Predefined are rendered above.
                if (hasMetric(predefinedMetrics[category], metric, labels.labels)) {
                    return <></>;
                }
                return (
                    <DataListItem>
                        <DataListItemRow>
                            <DataListItemCells
                                dataListCells={[
                                    onCustomChange ? (
                                        <DataListCheck
                                            id={metric}
                                            aria-labelledby={metric}
                                            name={metric}
                                            isChecked
                                            isDisabled
                                        />
                                    ) : (
                                        <></>
                                    ),
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

export function PrometheusMetricsCard(
    category: Category,
    period: number,
    mcfg: Record<string, Labels> | undefined,
    title: string
) {
    const hasMetrics = mcfg && Object.keys(mcfg).length > 0;
    return (
        <GridItem md={hasMetrics ? 12 : 6} lg={hasMetrics ? 12 : 6}>
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
                                <DescriptionListGroup>
                                    <DescriptionListTerm>Gathering period</DescriptionListTerm>
                                    <DescriptionListDescription>
                                        {period}&nbsp;
                                        {pluralize('minute', period)}
                                    </DescriptionListDescription>
                                </DescriptionListGroup>
                            ) : (
                                <></>
                            )}
                            <DescriptionListGroup>
                                <DescriptionListTerm>Metrics</DescriptionListTerm>
                                <DescriptionListDescription>
                                    {prometheusMetricsDataList(mcfg, category, undefined)}
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
    category: Category,
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

export function PrometheusMetricsForm(
    pcfg: PrivateConfig,
    category: Category,
    title: string,
    onChange: (value, event) => Promise<void> | Promise<FormikErrors<FormikValues>>,
    onCustomChange:
        | ((value: unknown, id: string) => Promise<void> | Promise<FormikErrors<FormikValues>>)
        | undefined
) {
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
                                <FormGroup
                                    label="Metrics configuration"
                                    fieldId={`privateConfig.metrics.${category}.metrics`}
                                >
                                    {prometheusMetricsDataList(
                                        pcfg?.metrics?.[category]?.metrics,
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
