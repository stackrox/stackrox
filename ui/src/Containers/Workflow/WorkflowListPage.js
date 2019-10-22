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
    query,
    queryOptions,
    defaultSorted,
    entityListType,
    getTableColumns,
    selectedRowId,
    idAttribute,
    SubComponent,
    showSubrows,
    search
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

    const { loading, error, data } = useQuery(query, queryOptions);

    if (isGQLLoading(loading, data)) return <Loader />;
    if (!data || !data.results || error) return <PageNotFound resourceType={entityListType} />;

    const tableColumns = getTableColumns(workflowState);
    const defaultExpandedRows = showSubrows ? getDefaultExpandedRows(data) : null;

    return (
        <EntityList
            entityType={entityListType}
            idAttribute={idAttribute}
            rowData={data.results}
            tableColumns={tableColumns}
            selectedRowId={selectedRowId}
            search={search}
            SubComponent={SubComponent}
            defaultSorted={defaultSorted}
            defaultExpanded={defaultExpandedRows}
            searchOptions={searchOptions}
        />
    );
};

WorkflowListPage.propTypes = {
    // eslint-disable-next-line
    query: PropTypes.any.isRequired,
    queryOptions: PropTypes.shape({}),
    defaultSorted: PropTypes.arrayOf(PropTypes.shape({})),
    entityListType: PropTypes.string.isRequired,
    getTableColumns: PropTypes.func.isRequired,
    entityContext: PropTypes.shape({}),
    selectedRowId: PropTypes.string,
    search: PropTypes.shape({}),
    SubComponent: PropTypes.func,
    showSubrows: PropTypes.bool,
    idAttribute: PropTypes.string
};

WorkflowListPage.defaultProps = {
    queryOptions: null,
    defaultSorted: [],
    entityContext: {},
    selectedRowId: null,
    search: null,
    SubComponent: null,
    showSubrows: false,
    idAttribute: 'id'
};

export default WorkflowListPage;
