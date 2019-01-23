import React from 'react';

import Widget from 'Components/Widget';
import StandardsAcrossEntity from 'Containers/Compliance2/widgets/StandardsAcrossEntity';
import StandardsByEntity from 'Containers/Compliance2/widgets/StandardsByEntity';
import EntityCompliance from 'Containers/Compliance2/widgets/EntityCompliance';
import Sunburst from 'Components/visuals/Sunburst';
import GaugeWithDetail from 'Components/visuals/GaugeWithDetail';

import {
    horizontalBarData,
    sunburstData,
    sunburstLegendData,
    multiGaugeData
} from 'mockData/graphDataMock';
import entityTypes from 'constants/entityTypes';

import DashboardHeader from './Header';

const ComplianceDashboardPage = () => (
    <section className="flex flex-1 flex-col h-full">
        <div className="flex flex-1 flex-col">
            <DashboardHeader />
            <div className="flex-1 relative bg-base-200 p-4 overflow-auto">
                <div className="grid xl:grid-columns-3 md:grid-columns-2 sm:grid-columns-1 grid-gap-6">
                    <div className="grid grid-columns-2 grid-gap-6">
                        <EntityCompliance type={entityTypes.CLUSTERS} />
                        <Widget header="Widget 2" className="bg-base-100">
                            Widget 2<br />
                            Widget 2<br />
                            Widget 2<br />
                        </Widget>
                        <Widget header="Widget 3" className="bg-base-100">
                            Widget 3<br />
                            Widget 3<br />
                            Widget 3<br />
                        </Widget>
                    </div>

                    <StandardsAcrossEntity type={entityTypes.CLUSTERS} data={horizontalBarData} />
                    <StandardsByEntity type={entityTypes.CLUSTERS} />

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

                    <Widget header="Compliance Across Clusters" className="bg-base-100">
                        <GaugeWithDetail data={multiGaugeData} dataProperty="passing" />
                    </Widget>

                    <StandardsAcrossEntity type={entityTypes.NAMESPACES} data={horizontalBarData} />

                    <StandardsAcrossEntity type={entityTypes.NODES} data={horizontalBarData} />
                </div>
            </div>
        </div>
    </section>
);

export default ComplianceDashboardPage;
