import React from 'react';
import PageNotFound from 'Components/PageNotFound';
import isGQLLoading from 'utils/gqlLoading';
import Loader from 'Components/Loader';
import { useQuery } from 'react-apollo';
import EntityList from 'Components/EntityList';
import PropTypes from 'prop-types';

const WorkflowEntityPage = ({
    query,
    queryOptions,
    defaultSorted,
    entityListType,
    getTableColumns,
    selectedRowId,
    idAttribute,
    search
}) => {
    const { loading, error, data } = useQuery(query, queryOptions);
    if (isGQLLoading(loading, data)) return <Loader />;

    if (!data || !data.results || error) return <PageNotFound resourceType={entityListType} />;

    const tableColumns = getTableColumns();

    return (
        <EntityList
            entityType={entityListType}
            idAttribute={idAttribute}
            rowData={data.results}
            tableColumns={tableColumns}
            selectedRowId={selectedRowId}
            search={search}
            defaultSorted={defaultSorted}
        />
    );
};

WorkflowEntityPage.propTypes = {
    // eslint-disable-next-line
    query: PropTypes.any.isRequired,
    queryOptions: PropTypes.shape({}),
    defaultSorted: PropTypes.arrayOf(PropTypes.shape({})),
    entityListType: PropTypes.string.isRequired,
    getTableColumns: PropTypes.func.isRequired,
    entityContext: PropTypes.shape({}),
    selectedRowId: PropTypes.string,
    search: PropTypes.shape({}),
    idAttribute: PropTypes.string
};

WorkflowEntityPage.defaultProps = {
    queryOptions: null,
    defaultSorted: [],
    entityContext: {},
    selectedRowId: null,
    search: null,
    idAttribute: 'id'
};

export default WorkflowEntityPage;
