import React from 'react';
import type { ReactElement } from 'react';

import { integrationsPath } from 'routePaths';

import NotFoundMessage from 'Components/NotFoundMessage';

const IntegrationsNotFoundPage = (): ReactElement => (
    <NotFoundMessage
        title="404: We couldn't find that page"
        message="Another page might have what you need, return to Integrations"
        actionText="Go to Integrations"
        url={integrationsPath}
    />
);

export default IntegrationsNotFoundPage;
