import React from 'react';
import { withRouter } from 'react-router-dom';
import ReactRouterPropTypes from 'react-router-prop-types';
import URLService from 'modules/URLService';
import { resourceTypes, standardTypes } from 'constants/entityTypes';

import StandardsByEntity from 'Containers/Compliance/widgets/StandardsByEntity';
import StandardsAcrossEntity from 'Containers/Compliance/widgets/StandardsAcrossEntity';
import ComplianceByStandard from 'Containers/Compliance/widgets/ComplianceByStandard';
import WaveBackground from 'images/wave-bg.svg';
import WaveBackground2 from 'images/wave-bg-2.svg';
import DashboardHeader from './Header';

const ComplianceDashboardPage = ({ match, location }) => {
    const params = URLService.getParams(match, location);

    return (
        <section className="flex flex-col relative min-h-full">
            <DashboardHeader
                classes="bg-gradient-horizontal z-10 sticky pin-t text-base-100"
                bgStyle={{
                    boxShadow: 'hsla(230, 75%, 63%, 0.62) 0px 5px 30px 0px',
                    '--start': 'hsl(226, 70%, 60%)',
                    '--end': 'hsl(226, 64%, 56%)'
                }}
                params={params}
            />
            <img
                className="absolute pin-l pointer-events-none z-10 w-full"
                src={WaveBackground2}
                style={{ mixBlendMode: 'lighten', top: '-60px' }}
                alt="Waves"
            />
            <div
                className="flex-1 relative bg-gradient-diagonal p-6 xxxl:p-8"
                style={{ '--start': '#d3d9ff', '--end': '#b9dbff' }}
                id="capture-dashboard"
            >
                <img
                    className="absolute pin-l pointer-events-none w-full"
                    src={WaveBackground}
                    style={{ top: '-130px' }}
                    alt="Wave"
                />
                <div
                    className="grid grid-gap-6 xxxl:grid-gap-8 md:grid-auto-fit xxl:grid-auto-fit-wide md:grid-dense"
                    style={{ '--min-tile-height': '160px' }}
                >
                    <StandardsAcrossEntity
                        type={resourceTypes.CLUSTER}
                        params={params}
                        bodyClassName="pr-4 py-1"
                        className="pdf-page"
                    />
                    <StandardsByEntity
                        type={resourceTypes.CLUSTER}
                        params={params}
                        bodyClassName="p-4"
                        className="pdf-page"
                    />
                    <StandardsAcrossEntity
                        type={resourceTypes.NAMESPACE}
                        params={params}
                        bodyClassName="px-4 pt-1"
                        className="pdf-page"
                    />
                    <StandardsAcrossEntity
                        type={resourceTypes.NODE}
                        params={params}
                        bodyClassName="pr-4 py-1"
                        className="pdf-page"
                    />
                    <ComplianceByStandard
                        type={standardTypes.CIS_DOCKER_V1_1_0}
                        params={params}
                        className="pdf-page"
                    />
                    <ComplianceByStandard
                        type={standardTypes.CIS_KUBERENETES_V1_2_0}
                        params={params}
                        className="pdf-page"
                    />
                    <ComplianceByStandard
                        type={standardTypes.HIPAA_164}
                        params={params}
                        className="pdf-page"
                    />
                    <ComplianceByStandard
                        type={standardTypes.NIST_800_190}
                        params={params}
                        className="pdf-page"
                    />
                    <ComplianceByStandard
                        type={standardTypes.PCI_DSS_3_2}
                        params={params}
                        className="pdf-page"
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
