import React, { useState } from 'react';
import { useHistory } from 'react-router-dom';
import { Chart, ChartAxis, ChartBar, ChartLabelProps } from '@patternfly/react-charts';

import useResizeObserver from 'hooks/useResizeObserver';
import {
    defaultChartHeight,
    defaultChartBarWidth,
    patternflySeverityTheme,
    navigateOnClickEvent,
    severityColorScale,
} from 'utils/chartUtils';
import { LinkableChartLabel } from 'Components/PatternFly/Charts/LinkableChartLabel';
import { SearchFilter } from 'types/search';
import { vulnManagementImagesPath } from 'routePaths';
import { getQueryString } from 'utils/queryStringUtils';
import isResourceScoped from '../utils';

// The available time buckets for this widget will always be a set of '4', so we can
// narrow the types to a tuple here for safer indexing throughout this component.
export type TimeRange = { enabled: boolean; value: number };
export type TimeRangeTuple = [TimeRange, TimeRange, TimeRange, TimeRange];
export const timeRangeTupleIndices = [0, 1, 2, 3] as const;
export type TimeRangeTupleIndex = (typeof timeRangeTupleIndices)[number];
export type TimeRangeCounts = Record<`timeRange${TimeRangeTupleIndex}`, number>;

export type ChartData = {
    barData: { x: string; y: number }[];
    labelLink: string;
    labelText: string;
    fill: string;
};

export type AgingImagesChartProps = {
    searchFilter: SearchFilter;
    timeRanges: TimeRangeTuple;
    timeRangeCounts: TimeRangeCounts;
};

export function getTimeFilterOption(ageRange: number, nextAgeRange?: number) {
    return typeof nextAgeRange === 'number' ? `${ageRange}d-${nextAgeRange}d` : `>${ageRange}d`;
}

function linkForAgingImages(searchFilter: SearchFilter, ageRange: number, nextAgeRange?: number) {
    const timeFilter = getTimeFilterOption(ageRange, nextAgeRange);
    const queryString = getQueryString({
        s: {
            ...searchFilter,
            'Image Created Time': timeFilter,
        },
        sort: [{ id: 'Image Created Time', desc: 'false' }],
    });
    return `${vulnManagementImagesPath}${queryString}`;
}

function yAxisTitle(searchFilter: SearchFilter) {
    return isResourceScoped(searchFilter) ? 'Active image count' : 'Image count';
}

// `datum` for these callbacks will refer to the index number of the bar in the chart. This index
// value matches the index of the target `ChartData` item passed to the chart component.
const labelLinkCallback = ({ datum }: ChartLabelProps, chartData: ChartData[]) => {
    return typeof datum === 'number' ? chartData[datum - 1]?.labelLink ?? '' : '';
};

const labelTextCallback = ({ datum }: ChartLabelProps, chartData: ChartData[]) => {
    return typeof datum === 'number' ? chartData[datum - 1]?.labelText ?? '' : '';
};

function makeChartData(
    searchFilter: SearchFilter,
    timeRanges: TimeRangeTuple,
    data: TimeRangeCounts
): ChartData[] {
    const chartData: ChartData[] = [];

    timeRangeTupleIndices.forEach((index) => {
        const { value, enabled } = timeRanges[index];

        if (enabled) {
            const nextEnabledRange = timeRanges.slice(index + 1).find((range) => range.enabled);
            const x = String(value);
            let y = data[`timeRange${index}`];
            // Since time ranges are grouped into buckets, we need to look forward and add the totals in any
            // disabled bucket to the current total.
            for (let i = index; i < timeRanges.length - 1; i += 1) {
                if (!timeRanges[i + 1].enabled) {
                    y += data[`timeRange${i + 1}`];
                } else {
                    break;
                }
            }
            const barData = [{ x, y }];
            const fill = severityColorScale[index];
            const labelLink = linkForAgingImages(searchFilter, value, nextEnabledRange?.value);
            let labelText: string;
            if (typeof nextEnabledRange === 'undefined') {
                // This is the last time range bucket
                labelText = value === 365 ? `>1 year` : `>${value} days`;
            } else {
                labelText = `${value}-${nextEnabledRange.value} days`;
            }

            chartData.push({ barData, fill, labelLink, labelText });
        }
    });

    return chartData;
}

function AgingImagesChart({ searchFilter, timeRanges, timeRangeCounts }: AgingImagesChartProps) {
    const history = useHistory();
    const [widgetContainer, setWidgetContainer] = useState<HTMLDivElement | null>(null);
    const widgetContainerResizeEntry = useResizeObserver(widgetContainer);
    const chartData = makeChartData(searchFilter, timeRanges, timeRangeCounts);

    return (
        <div ref={setWidgetContainer}>
            <Chart
                ariaDesc="Aging images grouped by date of last update"
                ariaTitle="Aging images"
                domainPadding={{ x: [50, 50] }}
                height={defaultChartHeight}
                width={widgetContainerResizeEntry?.contentRect.width} // Victory defaults to 450
                padding={{
                    top: 25,
                    left: 65,
                    right: 10,
                    bottom: 60,
                }}
                theme={patternflySeverityTheme}
            >
                <ChartAxis
                    label="Image age"
                    tickLabelComponent={
                        <LinkableChartLabel
                            linkWith={(props) => labelLinkCallback(props, chartData)}
                            text={(props) => labelTextCallback(props, chartData)}
                        />
                    }
                />
                <ChartAxis
                    label={yAxisTitle(searchFilter)}
                    tickFormat={String}
                    style={{
                        axisLabel: { padding: 50 },
                    }}
                    dependentAxis
                    showGrid
                />
                {chartData.map(({ barData, fill, labelLink }) => {
                    return (
                        <ChartBar
                            key={fill}
                            barWidth={defaultChartBarWidth}
                            data={barData}
                            labels={({ datum }) => `${Math.round(parseInt(datum.y, 10))}`}
                            style={{ data: { fill } }}
                            events={[navigateOnClickEvent(history, () => labelLink)]}
                        />
                    );
                })}
            </Chart>
        </div>
    );
}

export default AgingImagesChart;
