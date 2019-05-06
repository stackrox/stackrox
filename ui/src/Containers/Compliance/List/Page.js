import React from 'react';
import PropTypes from 'prop-types';
import { withRouter } from 'react-router-dom';
import ReactRouterPropTypes from 'react-router-prop-types';
import URLService from 'modules/URLService';
import entityTypes, { standardBaseTypes } from 'constants/entityTypes';
import { standardLabels } from 'messages/standards';
import CollapsibleBanner from 'Components/CollapsibleBanner/CollapsibleBanner';
import ComplianceAcrossEntities from 'Containers/Compliance/widgets/ComplianceAcrossEntities';
import ControlsMostFailed from 'Containers/Compliance/widgets/ControlsMostFailed';
import ComplianceList from 'Containers/Compliance/List/List';
import SearchInput from '../SearchInput';
import Header from './Header';

const ComplianceListPage = ({ match, location }) => {
    const params = URLService.getParams(match, location);
    const groupBy = params.query && params.query.groupBy ? params.query.groupBy : null;
    let { entityType } = params;
    const query = { ...params.query };

    // TODO: get rid of this when standards have their own path.
    if (standardBaseTypes[entityType]) {
        query.standard = standardLabels[entityType];
        query.standardId = entityType;
        entityType = entityTypes.CONTROL;
    }

    const collapsibleBanner =
        entityTypes.CONTROL === entityType ? null : (
            <CollapsibleBanner className="pdf-page">
                <ComplianceAcrossEntities entityType={entityType} query={query} groupBy={groupBy} />
                <ControlsMostFailed entityType={entityType} query={query} showEmpty />
            </CollapsibleBanner>
        );

    return (
        <section className="flex flex-col h-full relative" id="capture-list">
            <Header
                searchComponent={
                    <SearchInput categories={['COMPLIANCE']} shouldAddComplianceState />
                }
            />
            {collapsibleBanner}
            <ComplianceList entityType={entityType} query={query} />
        </section>
    );
};

ComplianceListPage.propTypes = {
    match: ReactRouterPropTypes.match.isRequired,
    location: ReactRouterPropTypes.location.isRequired,
    params: PropTypes.shape({
        entityType: PropTypes.string.isRequired
    })
};

ComplianceListPage.defaultProps = {
    params: null
};

export default withRouter(ComplianceListPage);
