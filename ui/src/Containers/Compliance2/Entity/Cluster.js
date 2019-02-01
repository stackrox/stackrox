import React from 'react';
import ComplianceByStandard from 'Containers/Compliance2/widgets/ComplianceByStandard';
import RelatedEntitiesList from 'Containers/Compliance2/widgets/RelatedEntitiesList';
import PropTypes from 'prop-types';
import entityTypes from 'constants/entityTypes';
import EntityCompliance from 'Containers/Compliance2/widgets/EntityCompliance';
import Widget from 'Components/Widget';
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
                        Widget 2<br />
                        Widget 2<br />
                        Widget 2<br />
                    </Widget>
                    <Widget header="Widget 3">
                        Widget 3<br />
                        Widget 3<br />
                        Widget 3<br />
                    </Widget>
                </div>
                {/* TO-DO: need to make sure these are the cluster widgets we want */}
                <ComplianceByStandard type={entityTypes.PCI} params={params} />
                <ComplianceByStandard type={entityTypes.NIST} params={params} />
                <ComplianceByStandard type={entityTypes.HIPAA} params={params} />
                <ComplianceByStandard type={entityTypes.CIS} params={params} />
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
