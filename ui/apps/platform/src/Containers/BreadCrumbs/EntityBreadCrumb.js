import React from 'react';
import { Link } from 'react-router-dom';
import upperFirst from 'lodash/upperFirst';
import pluralize from 'pluralize';

import entityLabels from 'messages/entity';
import useEntityName from 'hooks/useEntityName';

const EntityBreadCrumb = ({ workflowEntity, url }) => {
    const { entityId, entityType } = workflowEntity;
    const typeLabel = entityLabels[entityType];
    const subTitle = entityId ? upperFirst(typeLabel) : 'Entity list';
    const { entityName } = useEntityName(entityType, entityId, !entityId);
    const title = entityName || pluralize(typeLabel);

    return (
        <span className="flex flex-col max-w-full" data-testid="breadcrumb-link-text">
            {url ? (
                <Link className="text-primary-700 underline font-700" title={title} to={url}>
                    {title}
                </Link>
            ) : (
                <span className="w-full truncate font-700" title={title}>
                    {title}
                </span>
            )}
            <span>{subTitle}</span>
        </span>
    );
};

export default EntityBreadCrumb;
