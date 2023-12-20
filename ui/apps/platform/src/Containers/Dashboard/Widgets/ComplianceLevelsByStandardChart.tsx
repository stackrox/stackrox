import React, { useState } from 'react';
import { useHistory } from 'react-router-dom';
import { Chart, ChartAxis, ChartBar, ChartGroup, ChartLabelProps } from '@patternfly/react-charts';

import { LinkableChartLabel } from 'Components/PatternFly/Charts/LinkableChartLabel';
import useResizeObserver from 'hooks/useResizeObserver';
import {
    defaultChartHeight,
    defaultChartBarWidth,
    navigateOnClickEvent,
    solidBlueChartColor,
    patternflySeverityTheme,
} from 'utils/chartUtils';

const labelLinkCallback = ({ datum }: ChartLabelProps, data: ComplianceData) => {
    return typeof datum === 'number' ? data[datum - 1]?.link ?? '' : '';
};

export type ComplianceData = {
    name: string;
    passing: number;
    link: string;
}[];

type ComplianceLevelsByStandardChartProps = {
    complianceData: ComplianceData;
};

function ComplianceLevelsByStandardChart({ complianceData }: ComplianceLevelsByStandardChartProps) {
    const history = useHistory();
    const [widgetContainer, setWidgetContainer] = useState<HTMLDivElement | null>(null);
    const widgetContainerResizeEntry = useResizeObserver(widgetContainer);

    return (
        <div ref={setWidgetContainer}>
            <Chart
                ariaDesc="Compliance coverage percentages by standard across the selected resource scope"
                ariaTitle="Compliance coverage by standard"
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
                            linkWith={(props) => labelLinkCallback(props, complianceData)}
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
                    {complianceData.map(({ name, passing, link }) => (
                        <ChartBar
                            key={name}
                            style={{ data: { fill: solidBlueChartColor } }}
                            barWidth={defaultChartBarWidth}
                            data={[{ x: name, y: passing, link }]}
                            labels={({ datum }) => `${Math.round(parseInt(datum.y, 10))}%`}
                            events={[navigateOnClickEvent(history, () => link)]}
                        />
                    ))}
                </ChartGroup>
            </Chart>
        </div>
    );
}

export default ComplianceLevelsByStandardChart;
