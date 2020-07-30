import { useQuery } from '@apollo/client';

import entityTypes from 'constants/entityTypes';
import { entityNameQueryMap } from 'utils/queryMap';
import getEntityName from 'utils/getEntityName';
import isGQLLoading from 'utils/gqlLoading';

function useEntityName(entityType, entityId, skip) {
    // Header query
    const entityNameQuery = entityNameQueryMap[entityType || entityTypes.CLUSTER];
    const nameQueryOptions = {
        options: {
            fetchPolicy: 'cache-first',
            skip,
        },
        variables: {
            id: decodeURIComponent(entityId) || '',
        },
    };
    const { loading, error, data } = useQuery(entityNameQuery, nameQueryOptions);

    let entityName;
    if (!isGQLLoading(loading, data)) {
        entityName = getEntityName(entityType, data);
    }

    return {
        loading,
        error,
        entityName,
    };
}

export default useEntityName;
