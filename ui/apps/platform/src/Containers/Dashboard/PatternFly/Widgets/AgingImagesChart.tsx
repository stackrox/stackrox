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
export type TimeRangeTupleIndex = typeof timeRangeTupleIndices[number];
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

function linkForAgingImages(searchFilter: SearchFilter, ageRange: number) {
    const queryString = getQueryString({
        s: {
            ...searchFilter,
            'Image Created Time': `>${ageRange}d`,
        },
        sort: [{ id: 'Image Created Time', desc: 'false' }],
    });
    return `${vulnManagementImagesPath}${queryString}`;
}

function yAxisTitle(searchFilter: SearchFilter) {
    return isResourceScoped(searchFilter) ? 'Active images' : 'All images';
}

// `datum` for these callbacks will refer to the index number of the bar in the chart. This index
// value matches the index of the target `ChartData` item passed to the chart component.
const labelLinkCallback = ({ datum }: ChartLabelProps, chartData: ChartData[]) => {
    return typeof datum === 'number' ? chartData[datum - 1].labelLink : '';
};

const labelTextCallback = ({ datum }: ChartLabelProps, chartData: ChartData[]) => {
    return typeof datum === 'number' ? chartData[datum - 1].labelText : '';
};

/**
 * Chart data is constructed from a 4-tuple of time range configurations and a data
 * object that contains image counts for each of the four time ranges.
 *
 * Since the incoming data contains overlapping image counts for each time range, we need
 * to process the data to group into buckets.
 *
 * The algorithm:
 * - Iterating over each time range, starting from the shortest.
 * - If the current time bucket is disabled, skip it.
 * - Find the next enabled time bucket after the current one
 * -- If one exists, subtract the count from the current bucket
 * -- If not, this is the last bucket so leave the count as-is
 * - Return the again image count, color, text, and link for this bucket
 */
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
            const nextEnabledIndex = timeRanges.findIndex((range) => range === nextEnabledRange);
            const x = String(value);
            const y =
                nextEnabledIndex !== -1
                    ? data[`timeRange${index}`] - data[`timeRange${nextEnabledIndex}`]
                    : data[`timeRange${index}`];
            const barData = [{ x, y }];
            const fill = severityColorScale[index];
            const labelLink = linkForAgingImages(searchFilter, value);
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
                animate={{ duration: 300 }}
                domainPadding={{ x: [50, 50] }}
                height={defaultChartHeight}
                width={widgetContainerResizeEntry?.contentRect.width} // Victory defaults to 450
                padding={{
                    top: 25,
                    left: 55,
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
                    padding={{ bottom: 10 }}
                    dependentAxis
                    showGrid
                />
                {chartData.map(({ barData, fill }) => {
                    return (
                        <ChartBar
                            key={fill}
                            barWidth={defaultChartBarWidth}
                            data={barData}
                            labels={({ datum }) => `${Math.round(parseInt(datum.y, 10))}`}
                            style={{ data: { fill } }}
                            events={[
                                navigateOnClickEvent(history, (targetProps) => {
                                    const range = targetProps?.datum?.xName;
                                    return linkForAgingImages(searchFilter, range);
                                }),
                            ]}
                        />
                    );
                })}
            </Chart>
        </div>
    );
}

export default AgingImagesChart;
