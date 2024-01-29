import React, { useState } from 'react';
import { Chart, ChartAxis, ChartBar, ChartGroup, ChartLabelProps } from '@patternfly/react-charts';

import { LinkableChartLabel } from 'Components/PatternFly/Charts/LinkableChartLabel';
import useResizeObserver from 'hooks/useResizeObserver';
import {
    defaultChartBarWidth,
    defaultChartHeight,
    patternflySeverityTheme,
} from 'utils/chartUtils';

import { getBarColor } from './ColorsForCompliance';
import { PassingRateData } from '../../types';

const labelLinkCallback = ({ datum }: ChartLabelProps, data: PassingRateData[]): string => {
    return typeof datum === 'number' ? data[datum - 1]?.link ?? '' : '';
};

type HorizontalBarChartProps = {
    passingRateData: PassingRateData[];
};

function HorizontalBarChart({ passingRateData }: HorizontalBarChartProps) {
    const [widgetContainer, setWidgetContainer] = useState<HTMLDivElement | null>(null);
    const widgetContainerResizeEntry = useResizeObserver(widgetContainer);

    return (
        <div ref={setWidgetContainer}>
            <Chart
                domainPadding={{ x: [20, 20] }}
                height={defaultChartHeight}
                width={widgetContainerResizeEntry?.contentRect.width}
                padding={{
                    top: 0,
                    left: 150,
                    right: 50,
                    bottom: 30,
                }}
                theme={patternflySeverityTheme}
            >
                <ChartAxis
                    tickLabelComponent={
                        <LinkableChartLabel
                            linkWith={(props) => labelLinkCallback(props, passingRateData)}
                        />
                    }
                />
                <ChartAxis
                    tickValues={[0, 50, 100]}
                    tickFormat={['0', '50%', '100%']}
                    padding={{ bottom: 10 }}
                    dependentAxis
                />
                <ChartGroup horizontal>
                    {passingRateData.map(({ name, passing }) => (
                        <ChartBar
                            key={name}
                            barWidth={defaultChartBarWidth}
                            data={[{ x: name, y: passing }]}
                            labels={({ datum }) => `${parseInt(datum.y, 10)}%`}
                            style={{
                                data: {
                                    fill: ({ datum }) => getBarColor(datum.y),
                                },
                            }}
                            sortOrder="ascending"
                        />
                    ))}
                </ChartGroup>
            </Chart>
        </div>
    );
}

export default HorizontalBarChart;
