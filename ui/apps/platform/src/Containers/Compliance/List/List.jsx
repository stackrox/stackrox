import React from 'react';
import PropTypes from 'prop-types';
import { useLocation, useNavigate } from 'react-router-dom';
import lowerCase from 'lodash/lowerCase';
import pluralize from 'pluralize';

import URLService from 'utils/URLService';
import { PageBody } from 'Components/Panel';
import SidePanelAdjacentArea from 'Components/SidePanelAdjacentArea';
import { searchCategories as searchCategoryTypes } from 'constants/entityTypes';
import searchContext from 'Containers/searchContext';
import { searchParams } from 'constants/searchParams';
import useWorkflowMatch from 'hooks/useWorkflowMatch';
import ListTable from './ListTable';
// TODO: this exception will be unnecessary once Compliance pages are re-structured like Config Management
/* eslint-disable-next-line import/no-cycle */
import SidePanel from './SidePanel';
import ComplianceSearchInput from '../ComplianceSearchInput';

const List = ({ entityType, query, selectedRowId, noSearch }) => {
    const navigate = useNavigate();
    const location = useLocation();
    const match = useWorkflowMatch();
    function setSelectedRowId(row) {
        const { id } = row;
        const url = URLService.getURL(match, location)
            .set('entityListType1', entityType)
            .set('entityId1', id)
            .url();
        navigate(url);
    }

    const placeholder = `Filter ${pluralize(lowerCase(entityType))}`;

    const searchComponent = noSearch ? null : (
        <ComplianceSearchInput
            placeholder={placeholder}
            categories={[searchCategoryTypes[entityType]]}
        />
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

export default List;

List.propTypes = {
    entityType: PropTypes.string.isRequired,
    query: PropTypes.shape({}),
    selectedRowId: PropTypes.string,
    // entityType2: PropTypes.string,
    // entityId2: PropTypes.string,
    noSearch: PropTypes.bool,
};

List.defaultProps = {
    query: null,
    selectedRowId: null,
    noSearch: false,
    // entityType2: null,
    // entityId2: null
};
