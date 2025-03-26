import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import {
    Chart,
    ChartAxis,
    ChartBar,
    ChartContainer,
    ChartGroup,
    ChartLabelProps,
} from '@patternfly/react-charts';

import { LinkableChartLabel } from 'Components/PatternFly/Charts/LinkableChartLabel';
import useResizeObserver from 'hooks/useResizeObserver';
import {
    defaultChartHeight,
    defaultChartBarWidth,
    navigateOnClickEvent,
    solidBlueChartColor,
    patternflySeverityTheme,
} from 'utils/chartUtils';

const labelLinkCallback = ({ datum }: ChartLabelProps, data: ComplianceLevelByStandard[]) => {
    return typeof datum === 'number' ? (data[datum - 1]?.link ?? '') : '';
};

export type ComplianceLevelByStandard = {
    name: string;
    passing: number;
    link: string;
};

type ComplianceLevelsByStandardChartProps = {
    complianceLevelsByStandard: ComplianceLevelByStandard[];
};

function ComplianceLevelsByStandardChart({
    complianceLevelsByStandard,
}: ComplianceLevelsByStandardChartProps) {
    const navigate = useNavigate();
    const [widgetContainer, setWidgetContainer] = useState<HTMLDivElement | null>(null);
    const widgetContainerResizeEntry = useResizeObserver(widgetContainer);

    return (
        <div ref={setWidgetContainer}>
            <Chart
                ariaDesc="Compliance coverage percentages by standard across the selected resource scope"
                ariaTitle="Compliance coverage by standard"
                containerComponent={<ChartContainer role="figure" />}
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
                            linkWith={(props) =>
                                labelLinkCallback(props, complianceLevelsByStandard)
                            }
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
                    {complianceLevelsByStandard.map(({ name, passing, link }) => (
                        <ChartBar
                            key={name}
                            style={{ data: { fill: solidBlueChartColor } }}
                            barWidth={defaultChartBarWidth}
                            data={[{ x: name, y: passing, link }]}
                            labels={({ datum }) => `${Math.round(parseInt(datum.y, 10))}%`}
                            events={[navigateOnClickEvent(navigate, () => link)]}
                        />
                    ))}
                </ChartGroup>
            </Chart>
        </div>
    );
}

export default ComplianceLevelsByStandardChart;
