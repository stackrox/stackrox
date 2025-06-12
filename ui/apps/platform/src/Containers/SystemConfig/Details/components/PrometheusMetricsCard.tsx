import React, { ReactElement } from 'react';
import { Category, Expression, Labels, PrivateConfig } from 'types/config.proto';
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
        image_vuln_namespace_severity: {
            labels: {
                Cluster: { expression: [] },
                Namespace: { expression: [] },
                Severity: { expression: [] },
            },
        },
        image_vuln_deployment_severity: {
            labels: {
                Cluster: { expression: [] },
                Namespace: { expression: [] },
                Deployment: { expression: [] },
                Severity: { expression: [] },
            },
        },
        image_vuln_user_workload_severity: {
            labels: {
                Cluster: { expression: [] },
                Namespace: { expression: [] },
                Deployment: { expression: [] },
                IsPlatformWorkload: {
                    expression: [{ operator: '=', argument: 'false' }],
                },
                Severity: { expression: [] },
            },
        },
    },
};

function labelGroup(labelExpression: Record<string, Expression>): ReactElement {
    return (
        <LabelGroup isCompact numLabels={Infinity}>
            {Object.entries(labelExpression).map(([label, expr]) => {
                return (
                    <Label isCompact key={label}>
                        {label}
                        {(expr?.expression || []).map((condition, i, arr) => (
                            <code>
                                {i > 0 &&
                                condition.operator !== 'OR' &&
                                arr[i - 1].operator !== 'OR'
                                    ? ' AND '
                                    : ' '}
                                {condition.operator}
                                {condition.argument}
                            </code>
                        ))}
                    </Label>
                );
            })}
        </LabelGroup>
    );
}

function predefinedMetricListItem(
    mcfg: Record<string, Labels> | undefined,
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
                                isChecked={mcfg && metric in mcfg}
                                onChange={(_, checked) =>
                                    onCustomChange(
                                        checked ? predefinedMetrics[category][metric] : null,
                                        `privateConfig.prometheusMetricsConfig.${category}.metrics.${metric}`
                                    )
                                }
                            />
                        ) : (
                            <></>
                        ),
                        <DataListCell>{metric}</DataListCell>,
                        <DataListCell>Predefined</DataListCell>,
                        <DataListCell>
                            {labelGroup(predefinedMetrics[category][metric].labels)}
                        </DataListCell>,
                    ]}
                />
            </DataListItemRow>
        </DataListItem>
    );
}

// isPredefined checks if the metric is one of the predefined ones by looking at
// the metric name, labels (ignoring the order) and expressions (ordered).
// NB: Returns false if the metric is not found in the actual configuration.
function isPredefined(
    category: Category,
    metric: string,
    mcfg: Record<string, Labels> | undefined
): boolean {
    if (!mcfg || !(metric in mcfg)) {
        return false;
    }
    const { labels } = mcfg[metric];
    if (!labels || !(metric in predefinedMetrics[category])) {
        return false;
    }
    const predefinedLabels = predefinedMetrics[category][metric].labels;
    return Object.entries(labels).every(([label, expr]) => {
        if (!(label in predefinedLabels)) {
            return false;
        }
        const predefinedExpr = predefinedLabels[label].expression || [];
        if (expr.expression?.length !== predefinedExpr.length) {
            return false;
        }
        return (
            predefinedExpr.length === 0 ||
            expr.expression.every(
                (condition, i) =>
                    condition.operator === predefinedExpr[i].operator &&
                    condition.argument === predefinedExpr[i].argument
            )
        );
    });
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
            {Object.keys(predefinedMetrics[category]).map((metric) => {
                // In view mode show only enabled predefined metrics.
                // In edit mode show all predefined metrics.
                if (onCustomChange || isPredefined(category, metric, mcfg)) {
                    return predefinedMetricListItem(mcfg, category, metric, onCustomChange);
                }
                return <></>;
            })}
            {Object.entries(mcfg || {}).map(([metric, labels]) => {
                // Predefined are rendered above.
                if (isPredefined(category, metric, mcfg)) {
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
                                    <DataListCell>{labelGroup(labels?.labels)}</DataListCell>,
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
    filter: string | undefined,
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
                            {filter ? (
                                <DescriptionListGroup>
                                    <DescriptionListTerm>Filter query</DescriptionListTerm>
                                    <DescriptionListDescription>
                                        {filter}
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

function prometheusMetricsFilterForm(
    pcfg: PrivateConfig,
    category: Category,
    onChange: (value, event) => Promise<void> | Promise<FormikErrors<FormikValues>>
): ReactElement {
    return (
        <FormGroup
            label="Filter query"
            fieldId={`privateConfig.prometheusMetricsConfig.${category}.filter`}
        >
            <TextInput
                isRequired
                type="search"
                id={`privateConfig.prometheusMetricsConfig.${category}.filter`}
                name={`privateConfig.prometheusMetricsConfig.${category}.filter`}
                value={pcfg?.prometheusMetricsConfig?.[category]?.filter}
                onChange={(event, value) => onChange(value, event)}
            />
        </FormGroup>
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
            fieldId={`privateConfig.prometheusMetricsConfig.${category}.gatheringPeriodMinutes`}
        >
            <TextInput
                isRequired
                type="number"
                id={`privateConfig.prometheusMetricsConfig.${category}.gatheringPeriodMinutes`}
                name={`privateConfig.prometheusMetricsConfig.${category}.gatheringPeriodMinutes`}
                value={pcfg?.prometheusMetricsConfig?.[category]?.gatheringPeriodMinutes || 0}
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
                                {prometheusMetricsFilterForm(pcfg, category, onChange)}
                            </GridItem>
                            <GridItem md={12}>
                                <FormGroup
                                    label="Metrics configuration"
                                    fieldId={`privateConfig.prometheusMetricsConfig.${category}.metrics`}
                                >
                                    {prometheusMetricsDataList(
                                        pcfg?.prometheusMetricsConfig?.[category]?.metrics,
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
