import React from 'react';
import PropTypes from 'prop-types';
import { withRouter } from 'react-router-dom';
import ReactRouterPropTypes from 'react-router-prop-types';
import URLService from 'modules/URLService';
import { searchCategories as searchCategoryTypes } from 'constants/entityTypes';
import ListTable from './Table';
import SidePanel from './SidePanel';
import SearchInput from '../SearchInput';

const ComplianceList = ({
    match,
    location,
    history,
    entityType,
    query,
    selectedRowId,
    noSearch
}) => {
    function setSelectedRowId(row) {
        const { id } = row;
        const url = URLService.getURL(match, location)
            .set('entityListType1', entityType)
            .set('entityId1', id)
            .url();
        history.push(url);
    }

    let sidepanel;
    if (selectedRowId) {
        sidepanel = <SidePanel entityType={entityType} entityId={selectedRowId} />;
    }
    const listQuery = Object.assign({}, query, URLService.getParams(match, location).query);
    const searchComponent = noSearch ? null : (
        <SearchInput categories={[searchCategoryTypes[entityType]]} />
    );
    return (
        <div className="flex flex-1 overflow-y-auto h-full bg-base-100">
            <ListTable
                searchComponent={searchComponent}
                selectedRowId={selectedRowId}
                entityType={entityType}
                query={listQuery}
                updateSelectedRow={setSelectedRowId}
                pdfId="capture-list"
            />
            {sidepanel}
        </div>
    );
};

export default withRouter(ComplianceList);

ComplianceList.propTypes = {
    entityType: PropTypes.string.isRequired,
    query: PropTypes.shape({}),
    selectedRowId: PropTypes.string,
    // entityType2: PropTypes.string,
    // entityId2: PropTypes.string,
    match: ReactRouterPropTypes.match.isRequired,
    history: ReactRouterPropTypes.history.isRequired,
    location: ReactRouterPropTypes.location.isRequired,
    noSearch: PropTypes.bool
};

ComplianceList.defaultProps = {
    query: null,
    selectedRowId: null,
    noSearch: false
    // entityType2: null,
    // entityId2: null
};
