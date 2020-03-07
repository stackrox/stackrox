import React from 'react';
import entityLabels from 'messages/entity';
import pluralize from 'pluralize';
import { Link } from 'react-router-dom';

import useEntityName from 'hooks/useEntityName';

const EntityBreadCrumb = ({ workflowEntity, url }) => {
    const { entityId, entityType } = workflowEntity;
    const typeLabel = entityLabels[entityType];
    const subTitle = entityId ? typeLabel : 'entity list';
    const { entityName } = useEntityName(entityType, entityId, !entityId);
    const title = entityName || pluralize(typeLabel);
    return (
        <span className="flex flex-col max-w-full" data-test-id="breadcrumb-link-text">
            {url ? (
                <Link
                    className="text-primary-700 underline uppercase truncate"
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
