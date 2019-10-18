import React from 'react';
import entityLabels from 'messages/entity';
import pluralize from 'pluralize';
import { Link } from 'react-router-dom';
import { entityNameQueryMap } from 'modules/queryMap';
import { useQuery } from 'react-apollo';
import getEntityName from 'modules/getEntityName';
import isGQLLoading from 'utils/gqlLoading';

const EntityBreadCrumb = ({ workflowEntity, url }) => {
    const { entityId, entityType } = workflowEntity;
    const typeLabel = entityLabels[entityType];
    let title = 'loading...';
    const subTitle = entityId ? typeLabel : 'entity list';

    const entityQuery = entityNameQueryMap[entityType];
    const queryOptions = {
        options: {
            fetchPolicy: 'cache-first',
            skip: !entityId
        },
        variables: {
            id: entityId
        }
    };
    const { loading, data } = useQuery(entityQuery, queryOptions);

    if (!isGQLLoading(loading, data)) {
        title = getEntityName(entityType, data) || pluralize(typeLabel);
    }

    return (
        <span className="flex flex-col max-w-full" data-test-id="breadcrumb-link-text">
            {url ? (
                <Link
                    className="text-primary-700 truncate uppercase truncate"
                    title={`${title}`}
                    to={url}
                >
                    {title}
                </Link>
            ) : (
                <span className="w-full truncate" title={title}>
                    <span className="truncate uppercase">{title}</span>
                </span>
            )}
            <span className="capitalize italic font-600">{subTitle}</span>
        </span>
    );
};

export default EntityBreadCrumb;
