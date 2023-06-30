import React, { ReactElement } from 'react';
import { Link } from 'react-router-dom';

import useEntityName from 'hooks/useEntityName';
import { VulnerabilityManagementEntityType } from 'utils/entityRelationships';

import {
    entityNounSentenceCasePlural,
    entityNounSentenceCaseSingular,
} from './entitiesForVulnerabilityManagement';

export type WorkflowEntity = {
    entityId: string;
    entityType: VulnerabilityManagementEntityType;
};

export type EntityBreadCrumbProps = {
    workflowEntity: WorkflowEntity;
    url: string | null;
};

function EntityBreadCrumb({ workflowEntity, url }: EntityBreadCrumbProps): ReactElement {
    const { entityId, entityType } = workflowEntity;
    const subTitle = entityId ? entityNounSentenceCaseSingular[entityType] : 'Entity list';
    const { entityName } = useEntityName(entityType, entityId, !entityId);
    const title = entityName || entityNounSentenceCasePlural[entityType];

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
}

export default EntityBreadCrumb;
