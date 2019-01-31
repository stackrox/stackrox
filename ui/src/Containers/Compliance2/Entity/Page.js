import React from 'react';
import ComplianceByStandard from 'Containers/Compliance2/widgets/ComplianceByStandard';
import RelatedEntitiesList from 'Containers/Compliance2/widgets/RelatedEntitiesList';
import URLService from 'modules/URLService';
import { withRouter } from 'react-router-dom';
import ReactRouterPropTypes from 'react-router-prop-types';
import PropTypes from 'prop-types';
import EntityCompliance from 'Containers/Compliance2/widgets/EntityCompliance';
import entityTypes from 'constants/entityTypes';
import Widget from 'Components/Widget';
import Header from './Header';

const ComplianceEntityPage = ({ match, location, params, sidePanelMode }) => {
    const widgetParams = sidePanelMode
        ? params
        : Object.assign({}, URLService.getParams(match, location), params);

    const EntityComplianceParams = Object.assign({}, widgetParams, {
        entityType: entityTypes.CLUSTERS
    });
    return (
        <section className="flex flex-col h-full w-full">
            {!sidePanelMode && <Header params={widgetParams} />}
            <div className="flex-1 relative bg-base-200 p-4 overflow-auto">
                <div
                    className={`grid ${
                        !sidePanelMode ? `xl:grid-columns-3 md:grid-columns-2` : ``
                    } sm:grid-columns-1 grid-gap-6`}
                >
                    <div className="grid grid-columns-2 grid-gap-6">
                        <EntityCompliance params={EntityComplianceParams} />
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
                    <ComplianceByStandard type={entityTypes.PCI} params={widgetParams} />
                    <ComplianceByStandard type={entityTypes.NIST} params={widgetParams} />
                    <ComplianceByStandard type={entityTypes.HIPAA} params={widgetParams} />
                    <ComplianceByStandard type={entityTypes.CIS} params={widgetParams} />
                    {!sidePanelMode && <RelatedEntitiesList params={widgetParams} />}
                </div>
            </div>
        </section>
    );
};

ComplianceEntityPage.propTypes = {
    match: ReactRouterPropTypes.match.isRequired,
    location: ReactRouterPropTypes.location.isRequired,
    params: PropTypes.shape({
        entityId: PropTypes.string,
        entityType: PropTypes.string
    }),
    sidePanelMode: PropTypes.bool
};

ComplianceEntityPage.defaultProps = {
    params: null,
    sidePanelMode: false
};

export default withRouter(ComplianceEntityPage);
