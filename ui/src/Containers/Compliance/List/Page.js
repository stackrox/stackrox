import React, { useContext } from 'react';
import PropTypes from 'prop-types';
import { withRouter } from 'react-router-dom';
import ReactRouterPropTypes from 'react-router-prop-types';
import URLService from 'modules/URLService';
import CollapsibleBanner from 'Components/CollapsibleBanner/CollapsibleBanner';
import ComplianceAcrossEntities from 'Containers/Compliance/widgets/ComplianceAcrossEntities';
import ControlsMostFailed from 'Containers/Compliance/widgets/ControlsMostFailed';
import ComplianceList from 'Containers/Compliance/List/List';
import searchContext from 'Containers/searchContext';
import SearchInput from '../SearchInput';
import Header from './Header';

const ComplianceListPage = ({ match, location }) => {
    const params = URLService.getParams(match, location);
    const searchParam = useContext(searchContext);
    const query = { ...params.query[searchParam] };
    const groupBy = query && query.groupBy ? query.groupBy : null;
    const { pageEntityListType, entityId1, entityType2, entityListType2, entityId2 } = params;
    return (
        <section className="flex flex-col h-full relative" id="capture-list">
            <Header
                entityType={pageEntityListType}
                searchComponent={
                    <SearchInput categories={['COMPLIANCE']} shouldAddComplianceState />
                }
            />
            <CollapsibleBanner className="pdf-page">
                <ComplianceAcrossEntities
                    entityType={pageEntityListType}
                    query={query}
                    groupBy={groupBy}
                />
                <ControlsMostFailed entityType={pageEntityListType} query={query} showEmpty />
            </CollapsibleBanner>
            <ComplianceList
                entityType={pageEntityListType}
                query={query}
                selectedRowId={entityId1}
                entityType2={entityType2}
                entityListType2={entityListType2}
                entityId2={entityId2}
                noSearch
            />
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
