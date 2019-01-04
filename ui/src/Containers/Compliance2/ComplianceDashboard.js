import React from 'react';
import PageHeader from 'Components/PageHeader';
import Panel from 'Components/Panel';
import HorizontalBarChart from 'Components/visuals/HorizontalBar';
import Sunburst from 'Components/visuals/Sunburst';
import VerticalBarChart from 'Components/visuals/VerticalClusterBar';
import {
    horizontalBarData,
    sunburstData,
    sunburstLegendData,
    verticalBarData
} from 'mockData/graphDataMock';

const ComplianceDashboard = () => {
    function formatAsPercent(x) {
        return `${x}%`;
    }
    return (
        <section className="flex flex-1 flex-col h-full">
            <div className="flex flex-1 flex-col">
                <PageHeader header="Compliance" subHeader="Dashboard" />
                <div className="flex-1 relative bg-base-200 p-4">
                    <div className="grid grid-columns-3 grid-gap-6">
                        <Panel
                            header="Standards Across Clusters"
                            className="widget"
                            bodyClassName="p-2"
                        >
                            <HorizontalBarChart
                                data={horizontalBarData}
                                valueFormat={formatAsPercent}
                            />
                        </Panel>

                        <Panel header="Standards By Cluster" className="widget">
                            <VerticalBarChart
                                data={verticalBarData}
                                labelLinks={{
                                    'Docker Swarm Dev': 'https://google.com/search?q=docker'
                                }}
                            />
                        </Panel>

                        <Panel header="Compliance Across Controls" className="widget">
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
                        </Panel>

                        <Panel header="Another Panel" className="widget">
                            More Graphs
                        </Panel>
                    </div>
                </div>
            </div>
        </section>
    );
};

export default ComplianceDashboard;
