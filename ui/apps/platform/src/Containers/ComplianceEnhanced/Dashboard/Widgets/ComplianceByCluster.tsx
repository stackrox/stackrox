import React, { useState } from 'react';
import { Chart, ChartAxis, ChartBar, ChartGroup, ChartLabelProps } from '@patternfly/react-charts';

import { LinkableChartLabel } from 'Components/PatternFly/Charts/LinkableChartLabel';
import useResizeObserver from 'hooks/useResizeObserver';
import {
    defaultChartHeight,
    defaultChartBarWidth,
    patternflySeverityTheme,
} from 'utils/chartUtils';

import {
    LOW_SEVERITY_COLOR,
    MODERATE_MEDIUM_SEVERITY_COLOR,
    IMPORTANT_HIGH_SEVERITY_COLOR,
    CRITICAL_SEVERITY_COLOR,
} from 'constants/visuals/colors';

import WidgetCard from '../../../Dashboard/Widgets/WidgetCard';

const labelLinkCallback = ({ datum }: ChartLabelProps, data: ComplianceData) => {
    return typeof datum === 'number' ? data[datum - 1]?.link ?? '' : '';
};

export type ComplianceData = {
    name: string;
    passing: number;
    link: string;
}[];

const mockComplianceData = [
    {
        name: 'staging',
        passing: 100,
        link: '',
    },
    {
        name: 'production',
        passing: 80,
        link: '',
    },
    {
        name: 'payments',
        passing: 73,
        link: '',
    },
    {
        name: 'patient-charts',
        passing: 69,
        link: '',
    },
    {
        name: 'another-cluster',
        passing: 67,
        link: '',
    },
    {
        name: 'cluster-name',
        passing: 39,
        link: '',
    },
];

function ComplianceByCluster() {
    const [complianceData] = useState(mockComplianceData);
    const [widgetContainer, setWidgetContainer] = useState<HTMLDivElement | null>(null);
    const widgetContainerResizeEntry = useResizeObserver(widgetContainer);

    function getBarColor(percent) {
        if (percent === 100) {
            return LOW_SEVERITY_COLOR;
        }
        if (percent > 50) {
            return MODERATE_MEDIUM_SEVERITY_COLOR;
        }
        if (percent > 25) {
            return IMPORTANT_HIGH_SEVERITY_COLOR;
        }
        return CRITICAL_SEVERITY_COLOR;
    }

    return (
        <WidgetCard isLoading={false} header="Compliance by cluster">
            <div ref={setWidgetContainer}>
                <Chart
                    ariaDesc="Compliance coverage percentages by cluster"
                    ariaTitle="Compliance coverage by cluster"
                    animate={{ duration: 300 }}
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
                        {complianceData.map(({ name, passing }) => (
                            <ChartBar
                                key={name}
                                barWidth={defaultChartBarWidth}
                                data={[{ x: name, y: passing }]}
                                labels={({ datum }) => `${Math.round(parseInt(datum.y, 10))}%`}
                                style={{
                                    data: {
                                        fill: ({ datum }) => getBarColor(datum.y),
                                    },
                                }}
                            />
                        ))}
                    </ChartGroup>
                </Chart>
            </div>
        </WidgetCard>
    );
}

export default ComplianceByCluster;
