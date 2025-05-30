import React, { ReactElement } from 'react';

import { MetricLabels, Expressions, PrivateConfig } from 'types/config.proto';
import {
    Card,
    CardBody,
    CardHeader,
    CardTitle,
    DescriptionList,
    DescriptionListDescription,
    DescriptionListGroup,
    DescriptionListTerm,
    Divider,
    Label,
} from '@patternfly/react-core';

function ivmetrics(privateConfig: PrivateConfig | null): ReactElement {
    let metrics: ReactElement[] = [];
    const mle: Map<string, MetricLabels> | undefined =
        privateConfig?.prometheusMetricsConfig?.imageVulnerabilities?.metricLabels;
    if (!mle) return <DescriptionListGroup></DescriptionListGroup>;
    for (const metric in mle) {
        const m: MetricLabels | undefined = mle[metric];
        if (!m) {
            continue;
        }
        const labelExprs: Map<string, Expressions> | undefined = m?.labelExpressions;
        if (!labelExprs) {
            continue;
        }
        const labels: string[] = [];
        for (const label in labelExprs) {
            labels.push(label);
        }
        metrics.push(
            <DescriptionListGroup>
                <DescriptionListTerm>{metric}</DescriptionListTerm>
                <DescriptionListDescription>{labels.join(', ')}</DescriptionListDescription>
            </DescriptionListGroup>
        );
    }
    return (
        <DescriptionList
            columnModifier={{
                default: '1Col',
            }}
        >
            {metrics}
        </DescriptionList>
    );
}

export type PrivateConfigPrometheusMetricsDetailsProps = {
    privateConfig: PrivateConfig;
};

const PrivateConfigPrometheusMetricsDetails = ({
    privateConfig,
}: PrivateConfigPrometheusMetricsDetailsProps): ReactElement => {
    const period =
        privateConfig?.prometheusMetricsConfig?.imageVulnerabilities?.gatheringPeriodHours;

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
                {
                    <>
                        <CardTitle component="h3">Image Vulnerabilities</CardTitle>
                    </>
                }
            </CardHeader>
            <Divider component="div" />
            <CardBody>
                <p className="pf-v5-u-mb-sm">
                    The discovered image vulnerabilities as Prometheus metrics.
                </p>
                <p className="pf-v5-u-mb-sm">
                    Gathered every {period} hour(s).
                </p>
                {ivmetrics(privateConfig)}
            </CardBody>
        </Card>
    );
};

export default PrivateConfigPrometheusMetricsDetails;
