import React from 'react';
import { horizontalBarData, sunburstData, sunburstLegendData } from 'mockData/graphDataMock';
import entityTypes from 'constants/entityTypes';

import Widget from 'Components/Widget';
import CollapsibleBanner from 'Components/CollapsibleBanner/CollapsibleBanner';
import StandardsAcrossEntity from 'Containers/Compliance2/widgets/StandardsAcrossEntity';
import StandardsByEntity from 'Containers/Compliance2/widgets/StandardsByEntity';
import Sunburst from 'Components/visuals/Sunburst';
import Header from './Header';

const ComplianceListPage = () => (
    <section className="flex flex-col h-full">
        <Header />
        <div className="flex-1 bg-base-200 overflow-auto">
            <CollapsibleBanner>
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
                <StandardsAcrossEntity type={entityTypes.NAMESPACES} data={horizontalBarData} />
                <StandardsAcrossEntity type={entityTypes.NODES} data={horizontalBarData} />
            </CollapsibleBanner>
        </div>
    </section>
);

export default ComplianceListPage;
