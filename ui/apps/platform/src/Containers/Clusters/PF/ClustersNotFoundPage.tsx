import React, { ReactElement } from 'react';

import { clustersListPath } from 'routePaths';

import NotFoundMessage from 'Components/NotFoundMessage';

const ClustersNotFoundPage = (): ReactElement => {
    return (
        <NotFoundMessage
            title="404: We couldn't find that cluster"
            message="The selected cluster doesn't exist or never existed. Return to the clusters page and selected a valid cluster."
            actionText="Go to Clusters"
            url={clustersListPath}
        />
    );
};

export default ClustersNotFoundPage;
