import React, { useState } from 'react';
import { Chart, ChartAxis, ChartBar, ChartContainer, ChartLabel } from '@patternfly/react-charts';
import { Bullseye, Spinner } from '@patternfly/react-core';
import { ExclamationCircleIcon } from '@patternfly/react-icons';

import EmptyStateTemplate from 'Components/EmptyStateTemplate';
import useResizeObserver from 'hooks/useResizeObserver';
import { ComplianceProfileScanStats } from 'services/ComplianceResultsStatsService';
import { defaultChartHeight, defaultChartBarWidth } from 'utils/chartUtils';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

import { getStatusCounts } from '../compliance.coverage.utils';

export type ProfileStatsWidgetProps = {
    isLoading: boolean;
    error: Error | undefined;
    profileScanStats: ComplianceProfileScanStats | undefined;
};

function ProfileStatsWidget({ error, isLoading, profileScanStats }: ProfileStatsWidgetProps) {
    const [widgetContainer, setWidgetContainer] = useState<HTMLDivElement | null>(null);
    const widgetContainerResizeEntry = useResizeObserver(widgetContainer);

    if (isLoading) {
        return (
            <Bullseye>
                <Spinner />
            </Bullseye>
        );
    }

    if (error) {
        return (
            <Bullseye>
                <EmptyStateTemplate
                    title="Error loading profile stats"
                    headingLevel="h3"
                    icon={ExclamationCircleIcon}
                    iconClassName="pf-v5-u-danger-color-100"
                >
                    {getAxiosErrorMessage(error.message)}
                </EmptyStateTemplate>
            </Bullseye>
        );
    }

    if (profileScanStats) {
        const { passCount, failCount, manualCount, otherCount, totalCount } = getStatusCounts(
            profileScanStats.checkStats
        );

        const data = [
            {
                x: 'Passing',
                y: passCount / totalCount,
                color: 'var(--pf-v5-global--primary-color--100)',
            },
            {
                x: 'Failing',
                y: failCount / totalCount,
                color: 'var(--pf-v5-global--danger-color--100)',
            },
            {
                x: 'Manual',
                y: manualCount / totalCount,
                color: 'var(--pf-v5-global--warning-color--100)',
            },
            {
                x: 'Mixed',
                y: otherCount / totalCount,
                color: 'var(--pf-v5-global--disabled-color--100)',
            },
        ];
        return (
            <div ref={setWidgetContainer}>
                <Chart
                    ariaDesc="Aging images grouped by date of last update"
                    ariaTitle="Aging images"
                    domainPadding={{ x: [50, 50] }}
                    padding={{ top: 25, bottom: 65, left: 65, right: 20 }}
                    height={defaultChartHeight}
                    width={widgetContainerResizeEntry?.contentRect.width}
                    containerComponent={<ChartContainer responsive />}
                >
                    <ChartAxis label="Check status" />
                    <ChartAxis
                        tickFormat={(t) => `${Math.round(t * 100)}%`}
                        domain={[0, 1]}
                        dependentAxis
                        showGrid
                    />
                    <ChartBar
                        key="testing"
                        barWidth={defaultChartBarWidth}
                        data={data}
                        style={{
                            data: {
                                fill: ({ datum }) => datum.color,
                            },
                        }}
                        labels={({ datum }) => `${Math.round(datum.y * 100)}%`}
                        labelComponent={<ChartLabel dy={-10} />}
                    />
                </Chart>
            </div>
        );
    }
}

export default ProfileStatsWidget;
