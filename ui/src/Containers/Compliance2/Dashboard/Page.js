import React from 'react';
import { withRouter } from 'react-router-dom';
import ReactRouterPropTypes from 'react-router-prop-types';
import URLService from 'modules/URLService';
import { resourceTypes } from 'constants/entityTypes';
import { sunburstData, sunburstLegendData } from 'mockData/graphDataMock';

import Sunburst from 'Components/visuals/Sunburst';
import Widget from 'Components/Widget';
import StandardsByEntity from 'Containers/Compliance2/widgets/StandardsByEntity';
import DashboardHeader from './Header';
import StandardsAcrossEntity from '../widgets/StandardsAcrossEntity';

const ComplianceDashboardPage = ({ match, location }) => {
    const params = URLService.getParams(match, location);

    return (
        <section className="flex flex-col h-full">
            <DashboardHeader params={params} />
            <div className="flex-1 relative bg-base-200 p-4 overflow-auto">
                <div className="grid xl:grid-columns-3 md:grid-columns-2 sm:grid-columns-1 grid-gap-6">
                    <StandardsAcrossEntity type={resourceTypes.CLUSTERS} params={params} />
                    <StandardsByEntity type={resourceTypes.CLUSTERS} params={params} />
                    <Widget header="Compliance Across Controls">
                        <Sunburst
                            data={sunburstData}
                            legendData={sunburstLegendData}
                            centerLabel="75%"
                        />
                    </Widget>
                    <StandardsAcrossEntity type={resourceTypes.NAMESPACES} params={params} />
                    <StandardsAcrossEntity type={resourceTypes.NODES} params={params} />
                    <Widget header="PCI Compliance">
                        <Sunburst
                            data={sunburstData}
                            legendData={sunburstLegendData}
                            centerLabel="75%"
                        />
                    </Widget>
                    <Widget header="NIST Compliance">
                        <Sunburst
                            data={sunburstData}
                            legendData={sunburstLegendData}
                            centerLabel="75%"
                        />
                    </Widget>
                    <Widget header="HIPAA Compliance">
                        <Sunburst
                            data={sunburstData}
                            legendData={sunburstLegendData}
                            centerLabel="75%"
                        />
                    </Widget>
                    <Widget header="CIS Compliance">
                        <Sunburst
                            data={sunburstData}
                            legendData={sunburstLegendData}
                            centerLabel="75%"
                        />
                    </Widget>
                </div>
            </div>
        </section>
    );
};

ComplianceDashboardPage.propTypes = {
    match: ReactRouterPropTypes.match.isRequired,
    location: ReactRouterPropTypes.location.isRequired
};

export default withRouter(ComplianceDashboardPage);
