import React, { useContext, useState } from 'react';
import PropTypes from 'prop-types';
import { useQuery } from '@apollo/client';
import { withRouter } from 'react-router-dom';
import ReactRouterPropTypes from 'react-router-prop-types';

import PageNotFound from 'Components/PageNotFound';
import Loader from 'Components/Loader';
import workflowStateContext from 'Containers/workflowStateContext';
import useFeatureFlags from 'hooks/useFeatureFlags';
import { SEARCH_OPTIONS_QUERY } from 'queries/search';
import { searchCategories as searchCategoryTypes } from 'constants/entityTypes';

import EntityList from './EntityList';

export function getDefaultExpandedRows(results) {
    return results
        ? results.map((_element, index) => {
              return { [index]: true };
          })
        : null;
}

const WorkflowListPage = ({
    data,
    totalResults,
    query,
    queryOptions,
    entityListType,
    getTableColumns,
    selectedRowId,
    idAttribute,
    SubComponent,
    showSubrows,
    sort,
    page,
    checkbox,
    tableHeaderComponents,
    selection,
    setSelection,
    renderRowActionButtons,
    history,
}) => {
    const workflowState = useContext(workflowStateContext);
    const [sortFields, setSortFields] = useState({});
    const { isFeatureFlagEnabled } = useFeatureFlags();

    const searchCategories = [searchCategoryTypes[entityListType]];
    const searchQueryOptions = {
        variables: {
            categories: searchCategories,
        },
    };
    const { data: searchData } = useQuery(SEARCH_OPTIONS_QUERY, searchQueryOptions);
    const searchOptions = (searchData && searchData.searchOptions) || [];

    const queryOptionsWithSkip = !data ? queryOptions : { skip: true };
    const { loading, error, data: ownQueryData } = useQuery(query, queryOptionsWithSkip);

    let displayData = data;
    let count = totalResults;
    const pageSize =
        queryOptions &&
        queryOptions.variables &&
        queryOptions.variables.pagination &&
        queryOptions.variables.pagination.limit;

    if (!data) {
        // @DEPRECATED, we no longer us the helper function isGQLLoading here,
        //    because now that we are using backend pagination
        //    it creates a weird UX lag where the table sort arrow changes,
        //    but there is no indication that we are waiting for more data from the backend
        if (loading) {
            return <Loader />;
        }

        if (!ownQueryData || !ownQueryData.results || error) {
            return <PageNotFound resourceType={entityListType} useCase={workflowState.useCase} />;
        }
        displayData = ownQueryData.results;
        count = ownQueryData.count;
    }

    const tableColumns = getTableColumns(workflowState, isFeatureFlagEnabled);
    const defaultExpandedRows = showSubrows ? getDefaultExpandedRows(displayData) : null;

    function onSortedChange(newSort, column) {
        const workflowSort = newSort.map((sortItem) => {
            const id = sortFields[sortItem.id] || column.sortField;
            setSortFields({ [sortItem.id]: id, ...sortFields });
            const { desc } = sortItem;
            return {
                id,
                desc,
            };
        });

        const url = workflowState.setSort(workflowSort).setPage(0).toUrl();
        history.push(url);
    }

    return (
        <EntityList
            entityType={entityListType}
            idAttribute={idAttribute}
            rowData={displayData}
            tableColumns={tableColumns}
            selectedRowId={selectedRowId}
            sort={sort}
            page={page}
            SubComponent={SubComponent}
            defaultExpanded={defaultExpandedRows}
            searchOptions={searchOptions}
            checkbox={checkbox}
            tableHeaderComponents={tableHeaderComponents}
            selection={selection}
            setSelection={setSelection}
            renderRowActionButtons={renderRowActionButtons}
            serverSidePagination
            onSortedChange={onSortedChange}
            totalResults={count}
            pageSize={pageSize}
            disableSortRemove
        />
    );
};

WorkflowListPage.propTypes = {
    query: PropTypes.shape({}),
    data: PropTypes.arrayOf(PropTypes.shape({})),
    queryOptions: PropTypes.shape({
        variables: PropTypes.shape({
            pagination: PropTypes.shape({
                limit: PropTypes.number,
            }),
        }),
    }),
    entityListType: PropTypes.string.isRequired,
    getTableColumns: PropTypes.func.isRequired,
    entityContext: PropTypes.shape({}),
    selectedRowId: PropTypes.string,
    sort: PropTypes.arrayOf(PropTypes.shape({})),
    page: PropTypes.number,
    SubComponent: PropTypes.func,
    showSubrows: PropTypes.bool,
    idAttribute: PropTypes.string,
    checkbox: PropTypes.bool,
    tableHeaderComponents: PropTypes.element,
    selection: PropTypes.arrayOf(PropTypes.string),
    setSelection: PropTypes.func,
    renderRowActionButtons: PropTypes.func,
    history: ReactRouterPropTypes.history.isRequired,
    totalResults: PropTypes.number,
};

WorkflowListPage.defaultProps = {
    query: null,
    queryOptions: null,
    data: null,
    entityContext: {},
    selectedRowId: null,
    sort: null,
    page: 0,
    SubComponent: null,
    showSubrows: false,
    idAttribute: 'id',
    checkbox: false,
    tableHeaderComponents: null,
    selection: [],
    setSelection: null,
    renderRowActionButtons: null,
    totalResults: 0,
};

export default withRouter(WorkflowListPage);
