import React from 'react';
import { Link } from 'react-router-dom';
import pluralize from 'pluralize';

import entityLabels from 'messages/entity';
import useEntityName from 'hooks/useEntityName';
import { shouldUseOriginalCase } from 'utils/workflowUtils';

const EntityBreadCrumb = ({ workflowEntity, url }) => {
    const { entityId, entityType } = workflowEntity;
    const typeLabel = entityLabels[entityType];
    const subTitle = entityId ? typeLabel : 'entity list';
    const { entityName } = useEntityName(entityType, entityId, !entityId);
    const title = entityName || pluralize(typeLabel);

    const useOriginalCase = shouldUseOriginalCase(entityName, entityType);
    const extraClasses = useOriginalCase ? '' : 'uppercase truncate';

    return (
        <span className="flex flex-col max-w-full" data-testid="breadcrumb-link-text">
            {url ? (
                <Link
                    className={`text-primary-700 underline ${extraClasses}`}
                    title={`${title}`}
                    to={url}
                >
                    {title}
                </Link>
            ) : (
                <span className="w-full truncate" title={title}>
                    <span className={extraClasses}>{title}</span>
                </span>
            )}
            <span className="capitalize font-600">{subTitle}</span>
        </span>
    );
};

export default EntityBreadCrumb;
