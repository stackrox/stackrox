import React, { ReactElement } from 'react';

import useIntegrations from './hooks/useIntegrations';
import usePageState from './hooks/usePageState';

import IntegrationPage from './IntegrationPage';
import IntegrationForm from './IntegrationForm';
import IntegrationsNotFoundPage from './IntegrationsNotFoundPage';

function IntegrationDetailsPage(): ReactElement {
    const {
        params: { source, type, id },
    } = usePageState();
    const integrations = useIntegrations({ source, type });
    const integration = integrations.find((datum) => datum.id === id);

    if (!integration) {
        return <IntegrationsNotFoundPage />;
    }

    return (
        <IntegrationPage
            title={integration.name}
            name={integration.name}
            traits={integration.traits}
        >
            <IntegrationForm source={source} type={type} initialValues={integration} />
        </IntegrationPage>
    );
}

export default IntegrationDetailsPage;
