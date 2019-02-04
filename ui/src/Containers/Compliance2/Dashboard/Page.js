import React from 'react';
import { withRouter } from 'react-router-dom';
import ReactRouterPropTypes from 'react-router-prop-types';
import URLService from 'modules/URLService';
import { resourceTypes, standardTypes } from 'constants/entityTypes';

import StandardsByEntity from 'Containers/Compliance2/widgets/StandardsByEntity';
import StandardsAcrossEntity from 'Containers/Compliance2/widgets/StandardsAcrossEntity';
import ComplianceByStandard from 'Containers/Compliance2/widgets/ComplianceByStandard';
import DashboardHeader from './Header';

const ComplianceDashboardPage = ({ match, location }) => {
    const params = URLService.getParams(match, location);

    return (
        <section className="flex flex-col h-full">
            <DashboardHeader params={params} />
            <div className="flex-1 relative bg-base-200 p-4 overflow-auto">
                <div className="grid xl:grid-columns-3 md:grid-columns-2 sm:grid-columns-1 grid-gap-6">
                    <StandardsAcrossEntity type={resourceTypes.CLUSTER} params={params} />
                    <StandardsByEntity type={resourceTypes.CLUSTER} params={params} />
                    <StandardsAcrossEntity type={resourceTypes.NAMESPACE} params={params} />
                    <StandardsAcrossEntity type={resourceTypes.NODE} params={params} />
                    <ComplianceByStandard type={standardTypes.PCI_DSS_3_2} params={params} />
                    <ComplianceByStandard type={standardTypes.NIST_800_190} params={params} />
                    <ComplianceByStandard type={standardTypes.HIPAA_164} params={params} />
                    <ComplianceByStandard type={standardTypes.CIS_DOCKER_V1_1_0} params={params} />
                    <ComplianceByStandard
                        type={standardTypes.CIS_KUBERENETES_V1_2_0}
                        params={params}
                    />
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
