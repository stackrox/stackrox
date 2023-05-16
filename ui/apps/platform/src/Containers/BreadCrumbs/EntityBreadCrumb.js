import React from 'react';
import { Link } from 'react-router-dom';
import capitalize from 'lodash/capitalize';
import pluralize from 'pluralize';

import entityLabels from 'messages/entity';
import useEntityName from 'hooks/useEntityName';

const EntityBreadCrumb = ({ workflowEntity, url }) => {
    const { entityId, entityType } = workflowEntity;
    const typeLabel = entityLabels[entityType];
    const subTitle = capitalize(entityId ? typeLabel : 'entity list');
    const { entityName } = useEntityName(entityType, entityId, !entityId);
    const title = entityName || pluralize(typeLabel);

    return (
        <span className="flex flex-col max-w-full" data-testid="breadcrumb-link-text">
            {url ? (
                <Link className="text-primary-700 underline" title={title} to={url}>
                    {title}
                </Link>
            ) : (
                <span className="w-full truncate" title={title}>
                    {title}
                </span>
            )}
            <span>{subTitle}</span>
        </span>
    );
};

export default EntityBreadCrumb;
