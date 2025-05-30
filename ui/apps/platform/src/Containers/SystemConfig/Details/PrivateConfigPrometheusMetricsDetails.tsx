import React, { ReactElement } from 'react';

import { MetricLabels, PrivateConfig, ImageVulnerabilities, Expressions } from 'types/config.proto';
import {
    Card,
    CardBody,
    CardHeader,
    CardTitle,
    Chip,
    ChipGroup,
    DescriptionList,
    DescriptionListDescription,
    DescriptionListGroup,
    DescriptionListTerm,
    Divider,
    Flex,
    FlexItem,
    Label,
    Toolbar,
} from '@patternfly/react-core';
import pluralize from 'pluralize';
import { Thead, Tr, Th, Td, Tbody, Table } from '@patternfly/react-table';

function buildLabelExpr(le: Record<string, Expressions>): ReactElement {
    let result: ReactElement[] = [];
    for (const label in le) {
        let e: ReactElement[] = [];
        for (const expr of le[label].expression) {
            e.push(<Chip isReadOnly>{expr.operator + expr.argument}</Chip>);
        }
        if (e.length) {
            result.push(<ChipGroup categoryName={label}>{e}</ChipGroup>);
        } else {
            result.push(<Chip isReadOnly>{label}</Chip>);
        }
    }
    return <ChipGroup>{result}</ChipGroup>;//<Toolbar>{result}</Toolbar>;
    //return <Toolbar><Flex className="search-filter-chips" spaceItems={{ default: 'spaceItemsXs' }}>{result}</Flex></Toolbar>;
}

function imageVulnerabilitiesMetrics(cfg: ImageVulnerabilities | undefined): ReactElement {
    const header: ReactElement[] = [];
    if (cfg?.query) {
        header.push(
            <DescriptionListGroup>
                <DescriptionListTerm>Filter query</DescriptionListTerm>
                <DescriptionListDescription>{cfg?.query}</DescriptionListDescription>
            </DescriptionListGroup>
        );
    }

    const mle: Record<string, MetricLabels> | undefined = cfg?.metricLabels;
    if (!mle) return <DescriptionListGroup>{header}</DescriptionListGroup>;
    return (
        <>
            <DescriptionListGroup>{header}</DescriptionListGroup>
            <Table variant="compact">
                <Thead>
                    <Tr>
                        <Th width={20}>Metric</Th>
                        <Th width={60}>Labels</Th>
                    </Tr>
                </Thead>

                <Tbody data-testid="integration-healths">
                    {Object.entries(mle).map(([metric, m]) => {
                        const labelExprs = m?.labelExpressions;
                        return (
                            <Tr key={metric}>
                                <Td
                                    dataLabel="Metric"
                                    modifier="breakWord"
                                    data-testid="metric-name"
                                >
                                    {metric}
                                </Td>
                                <Td
                                    dataLabel="Labels"
                                    modifier="breakWord"
                                    data-testid="metric-labels"
                                >
                                    {buildLabelExpr(labelExprs)}
                                </Td>
                            </Tr>
                        );
                    })}
                </Tbody>
            </Table>
        </>
    );

    /*
    for (const metric in mle) {
        const m: MetricLabels | undefined = mle[metric];
        if (!m) {
            </Tbody>continue;
        }
        const labelExprs: Map<string, Expressions> | undefined = m?.labelExpressions;
        if (!labelExprs) {
            continue;
        }
        metrics.push(<DescriptionListTerm>Metric:&nbsp;{metric}</DescriptionListTerm>);
        metrics.push(
            <DescriptionListDescription>
                Labels:&nbsp;{Object.keys(labelExprs).join(', ')}
            </DescriptionListDescription>
        );
    }
    return <DescriptionListGroup>{header}</DescriptionListGroup>;
    */
}

function imageVulnerabilitiesCard(cfg: ImageVulnerabilities | undefined) {
    const period = cfg?.gatheringPeriodHours;
    return (
        <Card isFlat data-testid="metrics-config">
            <CardHeader
                actions={{
                    actions: (
                        <>
                            {period ? (
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
                <CardTitle component="h3">Image Vulnerabilities</CardTitle>
            </CardHeader>
            <Divider component="div" />
            <CardBody>
                <DescriptionList
                    columnModifier={{
                        default: '1Col',
                    }}
                >
                    <DescriptionListGroup>
                        <DescriptionListTerm>Gathering period</DescriptionListTerm>
                        <DescriptionListDescription>
                            <span className="pf-v5-u-font-size-xl pf-v5-u-font-weight-bold">
                                {cfg?.gatheringPeriodHours}&nbsp;
                                {pluralize('hour', cfg?.gatheringPeriodHours)}
                            </span>
                        </DescriptionListDescription>
                    </DescriptionListGroup>
                    {imageVulnerabilitiesMetrics(cfg)}
                </DescriptionList>
            </CardBody>
        </Card>
    );
}

export type PrivateConfigPrometheusMetricsDetailsProps = {
    privateConfig: PrivateConfig;
};

const PrivateConfigPrometheusMetricsDetails = ({
    privateConfig,
}: PrivateConfigPrometheusMetricsDetailsProps): ReactElement => {
    const imageVulnerabilitiesCfg = privateConfig?.prometheusMetricsConfig?.imageVulnerabilities;

    return imageVulnerabilitiesCard(imageVulnerabilitiesCfg);
};

export default PrivateConfigPrometheusMetricsDetails;
