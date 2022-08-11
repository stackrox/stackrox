import React from 'react';

import { ChartDonut } from '@patternfly/react-charts';
import { Card, CardBody, CardHeader, CardTitle } from '@patternfly/react-core';
import PluginProvider from 'console-plugins/PluginProvider';
import { patternflySeverityTheme, severityColorScale } from 'utils/chartUtils';
import { gql, useQuery } from '@apollo/client';

function Comp() {
    const query = gql`
        query q {
            Critical: violationCount(query: "Severity:CRITICAL_SEVERITY+Violation Time:<1d")
            High: violationCount(query: "Severity:HIGH_SEVERITY+Violation Time:<1d")
            Medium: violationCount(query: "Severity:MEDIUM_SEVERITY+Violation Time:<1d")
            Low: violationCount(query: "Severity:LOW_SEVERITY+Violation Time:<1d")
        }
    `;
    const {
        data = {
            Critical: 0,
            High: 0,
            Medium: 0,
            Low: 0,
        },
    } = useQuery(query);

    return (
        <Card>
            <CardHeader>
                <CardTitle>Severity of violations today</CardTitle>
            </CardHeader>
            <CardBody>
                <div style={{ width: '300px' }}>
                    <ChartDonut
                        ariaDesc="Average number of pets"
                        ariaTitle="Donut chart example"
                        constrainToVisibleArea
                        data={Object.entries(data)
                            .reverse()
                            .map(([x, y]) => ({ x, y }))}
                        labels={({ datum }) => `${datum.x as string}: ${datum.y as string}%`}
                        legendData={Object.entries(data)
                            .reverse()
                            .map(([x, y]) => ({
                                name: `${x}: ${y as string}`,
                            }))}
                        legendOrientation="vertical"
                        legendPosition="right"
                        padding={{
                            right: 140, // Adjusted to accommodate legend
                        }}
                        subTitle="Violations today"
                        title={Object.entries(data)
                            .reduce((total, [, y]) => total + (y as number), 0)
                            .toString()}
                        theme={patternflySeverityTheme}
                        colorScale={severityColorScale}
                        width={350}
                    />
                </div>
            </CardBody>
        </Card>
    );
}

export default function DashboardRecent() {
    return (
        <PluginProvider>
            <Comp />
        </PluginProvider>
    );
}
