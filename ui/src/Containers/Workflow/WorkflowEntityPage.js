import React from 'react';
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
    getListData
}) => {
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
            entityContext={{ ...entityContext, [entityType]: entityId }}
        />
    ) : (
        <OverviewComponent data={result} entityContext={entityContext} />
    );
};

// Page.propTypes = {
//     location: ReactRouterPropTypes.location
// };

// Page.defaultProps = {
//     location: null
// };

export default WorkflowEntityPage;
