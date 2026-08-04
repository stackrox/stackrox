import type { ReactElement } from 'react';
import { Link } from 'react-router-dom-v5-compat';

import { riskWorkloadsBasePath } from 'routePaths';
import { getQueryString } from 'utils/queryStringUtils';
import type { AdministrationEventResource } from 'services/AdministrationEventsService';

function getResourceLink(resource: AdministrationEventResource): string | null {
    const { type, name } = resource;

    if (type === 'Image' && name) {
        return `${riskWorkloadsBasePath}${getQueryString({
            s: { Image: name },
            filteredWorkflowView: 'Full view',
        })}`;
    }

    return null;
}

export type AdministrationEventResourceLinkProps = {
    resource: AdministrationEventResource;
};

function AdministrationEventResourceLink({
    resource,
}: AdministrationEventResourceLinkProps): ReactElement | null {
    const { name } = resource;

    if (!name) {
        return null;
    }

    const link = getResourceLink(resource);

    if (link) {
        return <Link to={link}>{name}</Link>;
    }

    return <>{name}</>;
}

export default AdministrationEventResourceLink;
