import React from 'react';
import ComplianceByStandard from 'Containers/Compliance2/widgets/ComplianceByStandard';
import RelatedEntitiesList from 'Containers/Compliance2/widgets/RelatedEntitiesList';
import PropTypes from 'prop-types';
import entityTypes from 'constants/entityTypes';
import EntityCompliance from 'Containers/Compliance2/widgets/EntityCompliance';
import Widget from 'Components/Widget';
import ResourceCount from 'Containers/Compliance2/widgets/ResourceCount';
import Header from './Header';

const ClusterPage = ({ sidePanelMode, params }) => (
    <section className="flex flex-col h-full w-full">
        {!sidePanelMode && <Header params={params} />}
        <div className="flex-1 relative bg-base-200 p-4 overflow-auto">
            <div
                className={`grid ${
                    !sidePanelMode ? `xl:grid-columns-3 md:grid-columns-2` : ``
                } sm:grid-columns-1 grid-gap-6`}
            >
                <div className="grid grid-columns-2 grid-gap-6">
                    <EntityCompliance params={params} />
                    <Widget header="Widget 2">
                        Widget x<br />
                        Widget 2<br />
                        Widget 2<br />
                    </Widget>
                    <ResourceCount type={entityTypes.NODE} params={params} />
                </div>
                {/* TO-DO: need to make sure these are the cluster widgets we want */}
                <ComplianceByStandard type={entityTypes.PCI_DSS_3_2} params={params} />
                <ComplianceByStandard type={entityTypes.NIST_800_190} params={params} />
                <ComplianceByStandard type={entityTypes.HIPAA_164} params={params} />
                <ComplianceByStandard type={entityTypes.CIS_KUBERENETES_V1_2_0} params={params} />
                <ComplianceByStandard type={entityTypes.CIS_DOCKER_V1_1_0} params={params} />
                {!sidePanelMode && <RelatedEntitiesList params={params} />}
            </div>
        </div>
    </section>
);

ClusterPage.propTypes = {
    sidePanelMode: PropTypes.bool,
    params: PropTypes.shape({}).isRequired
};

ClusterPage.defaultProps = {
    sidePanelMode: false
};

export default ClusterPage;
