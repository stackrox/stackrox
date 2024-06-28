import React, { useState } from 'react';
import { Chart, ChartAxis, ChartBar, ChartContainer, ChartLabel } from '@patternfly/react-charts';
import { Bullseye, Spinner } from '@patternfly/react-core';
import { ExclamationCircleIcon } from '@patternfly/react-icons';

import EmptyStateTemplate from 'Components/EmptyStateTemplate';
import useResizeObserver from 'hooks/useResizeObserver';
import { ComplianceProfileScanStats } from 'services/ComplianceResultsStatsService';
import { defaultChartHeight, defaultChartBarWidth } from 'utils/chartUtils';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

import {
    FAILING_VAR_COLOR,
    MANUAL_VAR_COLOR,
    OTHER_VAR_COLOR,
    PASSING_VAR_COLOR,
} from '../compliance.coverage.constants';
import { getStatusCounts } from '../compliance.coverage.utils';

type ChartData = {
    x: string;
    y: number;
    color: string;
};

type DatumArgs = {
    datum?: ChartData;
};

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

        const data: ChartData[] = [
            {
                x: 'Passing',
                y: passCount / totalCount,
                color: PASSING_VAR_COLOR,
            },
            {
                x: 'Failing',
                y: failCount / totalCount,
                color: FAILING_VAR_COLOR,
            },
            {
                x: 'Manual',
                y: manualCount / totalCount,
                color: MANUAL_VAR_COLOR,
            },
            {
                x: 'Other',
                y: otherCount / totalCount,
                color: OTHER_VAR_COLOR,
            },
        ];

        return (
            <div ref={setWidgetContainer}>
                <Chart
                    ariaDesc="Percentage of total checks by status"
                    ariaTitle="Check stats by status"
                    domainPadding={{ x: [50, 50] }}
                    padding={{ top: 30, bottom: 65, left: 65, right: 20 }}
                    height={defaultChartHeight}
                    width={widgetContainerResizeEntry?.contentRect.width ?? 0}
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
                        barWidth={defaultChartBarWidth}
                        data={data}
                        style={{
                            data: {
                                fill: ({ datum }: DatumArgs) =>
                                    datum ? datum.color : OTHER_VAR_COLOR,
                            },
                        }}
                        labels={({ datum }: DatumArgs) =>
                            datum ? `${Math.round(datum.y * 100)}%` : ''
                        }
                        labelComponent={<ChartLabel dy={-10} />}
                    />
                </Chart>
            </div>
        );
    }
}

export default ProfileStatsWidget;
