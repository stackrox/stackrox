import React from 'react';

import Widget from 'Components/Widget';
import StandardsAcrossEntity from 'Containers/Compliance2/widgets/StandardsAcrossEntity';
import Sunburst from 'Components/visuals/Sunburst';
import VerticalBarChart from 'Components/visuals/VerticalClusterBar';

import {
    horizontalBarData,
    sunburstData,
    sunburstLegendData,
    verticalBarData
} from 'mockData/graphDataMock';
import entityTypes from 'constants/entityTypes';

import DashboardHeader from './Header';

const ComplianceDashboardPage = () => (
    <section className="flex flex-1 flex-col h-full">
        <div className="flex flex-1 flex-col">
            <DashboardHeader />
            <div className="flex-1 relative bg-base-200 p-4 overflow-auto">
                <div className="grid xl:grid-columns-3 md:grid-columns-2 sm:grid-columns-1 grid-gap-6">
                    <StandardsAcrossEntity type={entityTypes.CLUSTERS} data={horizontalBarData} />

                    <Widget header="Standards By Cluster" className="bg-base-100">
                        <VerticalBarChart
                            data={verticalBarData}
                            labelLinks={{
                                'Docker Swarm Dev': 'https://google.com/search?q=docker'
                            }}
                        />
                    </Widget>

                    <Widget header="Compliance Across Controls" className="bg-base-100">
                        <Sunburst
                            data={sunburstData}
                            legendData={sunburstLegendData}
                            centerLabel="75%"
                            containerProps={{
                                style: {
                                    borderRight: '1px solid var(--base-300)'
                                }
                            }}
                        />
                    </Widget>

                    <StandardsAcrossEntity type={entityTypes.NAMESPACES} data={horizontalBarData} />

                    <StandardsAcrossEntity type={entityTypes.NODES} data={horizontalBarData} />
                </div>
            </div>
        </div>
    </section>
);

export default ComplianceDashboardPage;
