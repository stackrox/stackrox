import React, { useContext } from 'react';
import PropTypes from 'prop-types';
import { useQuery } from 'react-apollo';

import PageNotFound from 'Components/PageNotFound';
import Loader from 'Components/Loader';
import EntityList from 'Components/EntityList';
import workflowStateContext from 'Containers/workflowStateContext';
import isGQLLoading from 'utils/gqlLoading';
import { SEARCH_OPTIONS_QUERY } from 'queries/search';
import { searchCategories as searchCategoryTypes } from 'constants/entityTypes';

export function getDefaultExpandedRows(data) {
    return data && data.results
        ? data.results.map((_element, index) => {
              return { [index]: true };
          })
        : null;
}

const WorkflowListPage = ({
    data,
    query,
    queryOptions,
    defaultSorted,
    entityListType,
    getTableColumns,
    selectedRowId,
    idAttribute,
    SubComponent,
    showSubrows,
    search,
    sort,
    page
}) => {
    const workflowState = useContext(workflowStateContext);

    const searchCategories = [searchCategoryTypes[entityListType]];
    const searchQueryOptions = {
        variables: {
            categories: searchCategories
        }
    };
    const { data: searchData } = useQuery(SEARCH_OPTIONS_QUERY, searchQueryOptions);
    const searchOptions = (searchData && searchData.searchOptions) || [];

    const queryOptionsWithSkip = !data ? queryOptions : { skip: true };
    const { loading, error, data: ownQueryData } = useQuery(query, queryOptionsWithSkip);

    let displayData = data;
    if (!data) {
        if (isGQLLoading(loading, ownQueryData)) return <Loader />;
        if (!ownQueryData || !ownQueryData.results || error)
            return <PageNotFound resourceType={entityListType} />;
        displayData = ownQueryData.results;
    }

    const tableColumns = getTableColumns(workflowState);
    const defaultExpandedRows = showSubrows ? getDefaultExpandedRows(data) : null;

    return (
        <EntityList
            entityType={entityListType}
            idAttribute={idAttribute}
            rowData={displayData}
            tableColumns={tableColumns}
            selectedRowId={selectedRowId}
            search={search}
            sort={sort}
            page={page}
            SubComponent={SubComponent}
            defaultSorted={defaultSorted}
            defaultExpanded={defaultExpandedRows}
            searchOptions={searchOptions}
        />
    );
};

WorkflowListPage.propTypes = {
    query: PropTypes.shape({}),
    data: PropTypes.arrayOf(PropTypes.shape({})),
    queryOptions: PropTypes.shape({
        options: PropTypes.shape({})
    }),
    defaultSorted: PropTypes.arrayOf(PropTypes.shape({})),
    entityListType: PropTypes.string.isRequired,
    getTableColumns: PropTypes.func.isRequired,
    entityContext: PropTypes.shape({}),
    selectedRowId: PropTypes.string,
    search: PropTypes.shape({}),
    sort: PropTypes.string,
    page: PropTypes.number,
    SubComponent: PropTypes.func,
    showSubrows: PropTypes.bool,
    idAttribute: PropTypes.string
};

WorkflowListPage.defaultProps = {
    query: null,
    queryOptions: null,
    data: null,
    defaultSorted: [],
    entityContext: {},
    selectedRowId: null,
    search: null,
    sort: null,
    page: 1,
    SubComponent: null,
    showSubrows: false,
    idAttribute: 'id'
};

export default WorkflowListPage;
