import React from 'react';
import PropTypes from 'prop-types';
import { useTheme } from 'Containers/ThemeProvider';

import PageNotFound from 'Components/PageNotFound';
import isGQLLoading from 'utils/gqlLoading';
import Loader from 'Components/Loader';
import { useQuery } from 'react-apollo';
import queryService from 'modules/queryService';
import getSubListFromEntity from 'utils/getSubListFromEntity';

const WorkflowEntityPage = ({
    ListComponent,
    OverviewComponent,
    entityType,
    entityId,
    entityListType,
    useCase,
    getListQuery,
    overviewQuery,
    queryOptions,
    entityContext,
    search,
    sort,
    page,
    getListData
}) => {
    const { isDarkMode } = useTheme();
    let query = overviewQuery;
    const getSubList = getListData || getSubListFromEntity;
    if (entityListType) {
        const { listFieldName, fragmentName, fragment } = queryService.getFragmentInfo(
            entityType,
            entityListType,
            useCase
        );
        query = getListQuery(listFieldName, fragmentName, fragment);
    }
    const { loading, data } = useQuery(query, queryOptions);
    if (isGQLLoading(loading, data)) return <Loader transparent />;
    if (!data || !data.result) return <PageNotFound resourceType={entityType} />;
    const { result } = data;

    const listData = entityListType ? getSubList(result, entityListType) : null;
    return entityListType ? (
        <ListComponent
            entityListType={entityListType}
            data={listData}
            search={search}
            sort={sort}
            page={page}
            entityContext={{ ...entityContext, [entityType]: entityId }}
        />
    ) : (
        <div
            className={`w-full flex ${
                !isDarkMode && !entityListType ? 'bg-side-panel-wave min-h-full' : 'h-full'
            }`}
        >
            <OverviewComponent data={result} entityContext={entityContext} />
        </div>
    );
};

WorkflowEntityPage.propTypes = {
    ListComponent: PropTypes.func.isRequired,
    OverviewComponent: PropTypes.func.isRequired,
    entityType: PropTypes.string.isRequired,
    entityId: PropTypes.string.isRequired,
    entityListType: PropTypes.string,
    useCase: PropTypes.string.isRequired,
    getListQuery: PropTypes.func.isRequired,
    overviewQuery: PropTypes.shape({}).isRequired,
    queryOptions: PropTypes.shape({}),
    entityContext: PropTypes.shape({}),
    search: PropTypes.shape({}),
    sort: PropTypes.string,
    page: PropTypes.number,
    getListData: PropTypes.func
};

WorkflowEntityPage.defaultProps = {
    entityListType: null,
    queryOptions: null,
    entityContext: {},
    search: null,
    sort: null,
    page: 1,
    getListData: null
};

export default WorkflowEntityPage;
