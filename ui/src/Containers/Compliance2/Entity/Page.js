import React from 'react';
import ComplianceByStandard from 'Containers/Compliance2/widgets/ComplianceByStandard';
import RelatedEntitiesList from 'Containers/Compliance2/widgets/RelatedEntitiesList';
import URLService from 'modules/URLService';
import { withRouter } from 'react-router-dom';
import ReactRouterPropTypes from 'react-router-prop-types';
import PropTypes from 'prop-types';
import Header from './Header';

const ComplianceEntityPage = ({ match, location, params, rowPageId }) => {
    const widgetParams = Object.assign({}, URLService.getParams(match, location), params);
    const urlPageId = URLService.getPageId(match);
    const pageId = rowPageId || urlPageId;

    const PCIWidgetParams = Object.assign({}, widgetParams, { standard: 'PCI' });
    const NISTWidgetParams = Object.assign({}, widgetParams, { standard: 'NIST' });
    const HIPAAWidgetParams = Object.assign({}, widgetParams, { standard: 'HIPAA' });
    const CISWidgetParams = Object.assign({}, widgetParams, { standard: 'CIS' });

    return (
        <section className="flex flex-col h-full w-full">
            {!rowPageId && <Header params={widgetParams} pageId={pageId} />}
            <div className="flex-1 relative bg-base-200 p-4 overflow-auto">
                <div
                    className={`grid ${
                        !rowPageId ? `xl:grid-columns-3 md:grid-columns-2` : ``
                    } sm:grid-columns-1 grid-gap-6`}
                >
                    <ComplianceByStandard params={PCIWidgetParams} pageId={pageId} />
                    <ComplianceByStandard params={NISTWidgetParams} pageId={pageId} />
                    <ComplianceByStandard params={HIPAAWidgetParams} pageId={pageId} />
                    <ComplianceByStandard params={CISWidgetParams} pageId={pageId} />
                    {!rowPageId && <RelatedEntitiesList params={widgetParams} pageId={pageId} />}
                </div>
            </div>
        </section>
    );
};

ComplianceEntityPage.propTypes = {
    match: ReactRouterPropTypes.match.isRequired,
    location: ReactRouterPropTypes.location.isRequired,
    params: PropTypes.shape({}),
    rowPageId: PropTypes.shape({})
};

ComplianceEntityPage.defaultProps = {
    params: null,
    rowPageId: null
};

export default withRouter(ComplianceEntityPage);
