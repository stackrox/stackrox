import React from 'react';
import PropTypes from 'prop-types';
import { withRouter } from 'react-router-dom';
import ReactRouterPropTypes from 'react-router-prop-types';
import URLService from 'utils/URLService';
import { PageBody } from 'Components/Panel';
import SidePanelAdjacentArea from 'Components/SidePanelAdjacentArea';
import { searchCategories as searchCategoryTypes } from 'constants/entityTypes';
import searchContext from 'Containers/searchContext';
import { searchParams } from 'constants/searchParams';
import ListTable from './Table';
// TODO: this exception will be unnecessary once Compliance pages are re-structured like Config Management
/* eslint-disable-next-line import/no-cycle */
import SidePanel from './SidePanel';
import SearchInput from '../SearchInput';

const ComplianceList = ({
    match,
    location,
    history,
    entityType,
    query,
    selectedRowId,
    noSearch,
}) => {
    function setSelectedRowId(row) {
        const { id } = row;
        const url = URLService.getURL(match, location)
            .set('entityListType1', entityType)
            .set('entityId1', id)
            .url();
        history.push(url);
    }

    const searchComponent = noSearch ? null : (
        <SearchInput categories={[searchCategoryTypes[entityType]]} />
    );

    return (
        <PageBody>
            <div className="flex-shrink-1 overflow-hidden w-full">
                <ListTable
                    searchComponent={searchComponent}
                    selectedRowId={selectedRowId}
                    entityType={entityType}
                    query={query}
                    updateSelectedRow={setSelectedRowId}
                    pdfId="capture-list"
                />
            </div>
            {selectedRowId && (
                <SidePanelAdjacentArea width="1/3">
                    <searchContext.Provider value={searchParams.sidePanel}>
                        <SidePanel entityType={entityType} entityId={selectedRowId} />
                    </searchContext.Provider>
                </SidePanelAdjacentArea>
            )}
        </PageBody>
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
    noSearch: PropTypes.bool,
};

ComplianceList.defaultProps = {
    query: null,
    selectedRowId: null,
    noSearch: false,
    // entityType2: null,
    // entityId2: null
};
