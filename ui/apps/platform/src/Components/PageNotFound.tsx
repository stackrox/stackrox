import React, { ReactElement } from 'react';

import { mainPath } from 'routePaths';

import NotFoundMessage from 'Components/NotFoundMessage';

export type PageNotFoundProps = {
    resourceType?: string;
    useCase?: string;
};

const PageNotFound = ({ resourceType = '', useCase = '' }: PageNotFoundProps): ReactElement => {
    const resourceTypeName = (resourceType || 'page').toLowerCase();
    const url = useCase ? `${mainPath}/${useCase}` : mainPath;
    const title = `Cannot find the ${resourceTypeName}`;
    const message =
        "It may have changed, did not exist, or no longer exists. Try using search from the dashboard to find what you're looking for.";

    return (
        <NotFoundMessage title={title} message={message} actionText="Go to dashboard" url={url} />
    );
};

export default PageNotFound;
